package main

import (
	"context"
	"encoding/json"
	"notes-memory-core-rag/internal/database"
	"notes-memory-core-rag/internal/handlers"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

type JobPayload struct {
	ID    string          `json:"id"`
	Type  string          `json:"type"`
	Input json.RawMessage `json:"input"`
}

const (
	maxRetries               = 3
	visibilityTimeoutMinutes = 3
)

var workerID string

func init() {
	workerID = uuid.New().String()[:8]
}

func main() {
	zlog.Logger = zlog.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	ctx := context.Background()

	// Load DB + Redis
	database.Connect()
	database.InitRedis()

	zlog.Info().Str("worker_id", workerID).Msg("⚙️ Worker Started - listening for jobs...")

	// Start background task to reclaim timed-out jobs
	go reclaimJobsTask(ctx)

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

		zlog.Info().Str("job_id", payload.ID).Str("worker_id", workerID).Msg("📥 Job received")

		// Process the job based on type
		switch payload.Type {
		case "query":
			processQueryJob(ctx, payload)
		default:
			zlog.Warn().Str("type", payload.Type).Msg("⚠️ Unknown job type")
		}
	}
}

func processQueryJob(ctx context.Context, job JobPayload) {
	zlog.Info().Str("job_id", job.ID).Str("worker_id", workerID).Msg("🤖 Processing query job")

	// Atomically claim the job with visibility timeout (prevents double processing)
	success, err := database.ClaimJobForProcessing(ctx, job.ID, workerID, visibilityTimeoutMinutes)
	if err != nil {
		zlog.Error().Err(err).Str("job_id", job.ID).Msg("Failed to claim job")
		return
	}

	if !success {
		zlog.Info().Str("job_id", job.ID).Msg("Job already being processed by another worker or timed out")
		return
	}

	// Convert raw input JSON into handler request struct
	var req handlers.QueryRequest
	if err := json.Unmarshal(job.Input, &req); err != nil {
		database.UpdateJobError(ctx, job.ID, "invalid job input")
		return
	}

	// Run the SAME logic /query handler uses with Retries
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// For long-running jobs, extend the visibility timeout
		if attempt > 1 {
			if err := database.ExtendVisibilityTimeout(ctx, job.ID, workerID, visibilityTimeoutMinutes); err != nil {
				zlog.Warn().Err(err).Str("job_id", job.ID).Msg("Failed to extend visibility timeout")
			}
		}

		ragResult, err := handlers.RunRAGPipeline(ctx, req.Query)
		if err == nil {
			database.UpdateJobResult(ctx, job.ID, ragResult)
			zlog.Info().
				Str("job_id", job.ID).
				Str("worker_id", workerID).
				Msg("✅ Job completed successfully")
			return
		}

		lastErr = err

		zlog.Warn().
			Int("attempt", attempt).
			Err(err).
			Str("worker_id", workerID).
			Msg("job execution failed, retrying")

		// Exponential backoff: 1s, 2s, 3s
		time.Sleep(time.Duration(1<<uint(attempt-1)) * time.Second)
	}

	database.UpdateJobError(ctx, job.ID, lastErr.Error())

	zlog.Error().
		Str("job_id", job.ID).
		Str("worker_id", workerID).
		Msg("❌ Job failed after retries")
}

// reclaimJobsTask runs in background to reclaim jobs that have timed out
func reclaimJobsTask(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			count, err := database.ReclaimTimedOutJobs(ctx)
			if err != nil {
				zlog.Error().Err(err).Msg("Failed to reclaim timed-out jobs")
				continue
			}

			if count > 0 {
				zlog.Info().
					Int("reclaimed_jobs", count).
					Str("worker_id", workerID).
					Msg("🔄 Reclaimed timed-out jobs")
			}

		case <-ctx.Done():
			zlog.Info().Msg("Stopping job reclaim task")
			return
		}
	}
}
