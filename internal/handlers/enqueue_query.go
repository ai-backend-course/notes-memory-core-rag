package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"notes-memory-core-rag/internal/database"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

func generateJobContentHash(req QueryRequest) string {
	normalized := struct {
		Query string `json:"query"`
	}{
		Query: strings.TrimSpace(strings.ToLower(req.Query)),
	}

	data, _ := json.Marshal(normalized)
	hash := sha256.Sum256(data)
	return hex.EncodeToString((hash[:]))
}

func EnqueueQueryJob(c *fiber.Ctx) error {
	ctx := context.Background()

	// Parse request body
	var req QueryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid input",
		})
	}

	//  Generate content hash
	var finalHash string
	// Use client-provided key
	if req.IdempotencyKey != nil {
		finalHash = "client:" + *req.IdempotencyKey
	} else {
		// Use content-based hash
		finalHash = "content:" + generateJobContentHash(req)
	}

	// Check for recent duplicate (last 5 minutes)
	existingJobID, err := database.CheckRecentDuplicateJob(ctx, finalHash, 5)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "duplicate check failed"})
	}

	if existingJobID != nil {
		// Return existing job instead of creating new one
		return c.JSON(fiber.Map{
			"job_id":  *existingJobID,
			"status":  "existing_job_found",
			"message": "Identical query was recently submitted",
		})
	}

	jobID, err := database.CreateJob(ctx, "query", req, finalHash)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create job record in database")
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create job record",
		})
	}

	// 3.  Prepare job payload for Redis
	jobPayload := map[string]interface{}{
		"id":    jobID,
		"type":  "query",
		"input": req,
	}

	payloadBytes, _ := json.Marshal(jobPayload)

	// 4. Push into Redis queue
	// Redis key: jobs:queue
	if database.RedisClient == nil {
		return c.Status(http.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "background jobs are not available on this deployment",
		})
	}

	if err := database.RedisClient.RPush(ctx, "jobs:queue", payloadBytes).Err(); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to enqueue job",
		})
	}

	// 5. Respond immediately
	return c.JSON(fiber.Map{
		"job_id": jobID,
		"status": "queued",
	})
}
