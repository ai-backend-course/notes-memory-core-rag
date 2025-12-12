package main

import (
	"context"
	"encoding/json"
	"notes-memory-core-rag/internal/database"
	"notes-memory-core-rag/internal/handlers"
	"os"
	"time"

	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

type JobPayload struct {
	ID    string          `json:"id"`
	Type  string          `json:"type"`
	Input json.RawMessage `json:"input"`
}

const (
	maxRetries = 3
)

func main() {
	// Pretty logs
	zlog.Logger = zlog.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	ctx := context.Background()

	// Load DB + Redis
	database.Connect()
	database.InitRedis()

	zlog.Info().Msg("⚙️ Worker Started - listening for jobs...")

	for {
		// BLPOP blocks until a job arrives
		result, err := database.RedisClient.BLPop(ctx, 0*time.Second, "jobs:queue").Result()
		if err != nil {
			zlog.Error().Err(err).Msg("BLPOP failed")
			continue
		}

		// result[0] = key
		// result[1] = JSON value
		raw := result[1]

		var payload JobPayload
		if err := json.Unmarshal([]byte(raw), &payload); err != nil {
			zlog.Error().Err(err).Msg("❌ Failed to unmarshal job payload")
			continue
		}

		zlog.Info().Str("job_id", payload.ID).Msg("📥 Job received")

		// Mark job as processing
		if err := database.UpdateJobStatus(ctx, payload.ID, "processing"); err != nil {
			zlog.Error().Err(err).Msg("❌ Failed to update status")
			continue
		}

		// PROCESS THE JOB BASED ON TYPE
		switch payload.Type {
		case "query":
			processQueryJob(ctx, payload)
		default:
			zlog.Warn().Str("type", payload.Type).Msg("⚠️ Unknown job type")
		}
	}
}

func processQueryJob(ctx context.Context, job JobPayload) {
	zlog.Info().Str("job_id", job.ID).Msg("🤖 Processing query job")

	// Convert raw input JSON into handler request struct
	var req handlers.QueryRequest
	if err := json.Unmarshal(job.Input, &req); err != nil {
		database.UpdateJobError(ctx, job.ID, "invalid job input")
		return
	}

	// Run the SAME logic /query handler uses with Retries
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		ragResult, err := handlers.RunRAGPipeline(ctx, req.Query)
		if err == nil {
			database.UpdateJobResult(ctx, job.ID, ragResult)
			zlog.Info().
				Str("job_id", job.ID).
				Msg("✅ Job completed successfully")
			return
		}

		lastErr = err

		zlog.Warn().
			Int("attempt", attempt).
			Err(err).
			Msg("job execution failed, retrying")

		// Exponential backoff: 1s, 2s, 3s
		time.Sleep(time.Duration(attempt) * time.Second)
	}

	database.UpdateJobError(ctx, job.ID, lastErr.Error())

	zlog.Error().
		Str("job_id", job.ID).
		Msg("❌ Job failed after retries")

}
