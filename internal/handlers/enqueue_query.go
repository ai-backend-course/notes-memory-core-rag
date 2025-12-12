package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"notes-memory-core-rag/internal/database"

	"github.com/gofiber/fiber/v2"
)

func EnqueueQueryJob(c *fiber.Ctx) error {
	ctx := context.Background()

	// 1. Parse request body
	var req QueryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid input",
		})
	}

	// 2. Create job in Postgres with status 'queued'
	jobID, err := database.CreateJob(ctx, "query", req)
	if err != nil {
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
