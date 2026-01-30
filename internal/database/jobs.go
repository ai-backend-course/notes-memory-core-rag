package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Job model
type Job struct {
	ID                string           `json:"id"`
	Type              string           `json:"type"`
	Input             json.RawMessage  `json:"input"`
	Status            string           `json:"status"`
	Result            *json.RawMessage `json:"result,omitempty"`
	Error             *string          `json:"error,omitempty"`
	VisibilityTimeout *time.Time       `json:"visibility_timeout,omitempty"`
	WorkerID          *string          `json:"worker_id,omitempty"`
}

// Create new job in DB - all jobs must have a content hash
func CreateJob(ctx context.Context, jobType string, input interface{}, contentHash string) (string, error) {
	id := uuid.New().String()

	// Convert input payload to JSON
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return "", err
	}

	// Insert into database with content_hash
	_, err = Pool.Exec(ctx, `
		INSERT INTO jobs (id, type, input, status, content_hash, retry_count)
		VALUES ($1, $2, $3, 'queued', $4, 0)
	`, id, jobType, inputBytes, contentHash)

	if err != nil {
		return "", err
	}

	return id, nil
}

// Update status
func UpdateJobStatus(ctx context.Context, id string, status string) error {
	_, err := Pool.Exec(ctx, `
		UPDATE jobs
	    SET status = $1, updated_at = NOW()
		WHERE id = $2
	`, status, id)
	return err
}

// Update result
func UpdateJobResult(ctx context.Context, id string, result interface{}) error {
	resultBytes, _ := json.Marshal(result)
	_, err := Pool.Exec(ctx, `
		UPDATE jobs
		SET status = 'completed',
			result = $1,
			visibility_timeout = NULL,
			worker_id = NULL,
			updated_at = NOW()
		WHERE id = $2
	`, resultBytes, id)
	return err
}

// Update error
func UpdateJobError(ctx context.Context, id string, errMsg string) error {
	_, err := Pool.Exec(ctx, `
		UPDATE jobs
		SET status = 'failed',
			error = $1,
			visibility_timeout = NULL,
			worker_id = NULL,
			updated_at = NOW()
		WHERE id = $2 
	`, errMsg, id)
	return err
}

// Fetch job by ID
func GetJobByID(ctx context.Context, id string) (*Job, error) {
	row := Pool.QueryRow(ctx, `
		SELECT id, type, input, status, result, error, visibility_timeout, worker_id
		FROM jobs
		WHERE id = $1
	`, id)

	var job Job
	err := row.Scan(
		&job.ID,
		&job.Type,
		&job.Input,
		&job.Status,
		&job.Result,
		&job.Error,
		&job.VisibilityTimeout,
		&job.WorkerID,
	)

	if err != nil {
		return nil, err
	}

	return &job, nil
}

// GetJobStatus returns just the status of a job by ID
func GetJobStatus(ctx context.Context, id string) (string, error) {
	var status string
	err := Pool.QueryRow(ctx, `
		SELECT status 
		FROM jobs 
		WHERE id = $1
	`, id).Scan(&status)

	if err != nil {
		return "", err
	}

	return status, nil
}

// ClaimJobForProcessing atomically updates job status from 'queued' to 'processing'
// with visibility timeout protection. Uses PostgreSQL's row-level locking to ensure
// only one worker can claim each job. Sets a visibility timeout to allow reclaim if worker crashes.
// Returns true if successfully claimed, false if already claimed by another worker
func ClaimJobForProcessing(ctx context.Context, id string, workerID string, timeoutMinutes int) (bool, error) {
	visibilityTimeout := time.Now().Add(time.Duration(timeoutMinutes) * time.Minute)

	result, err := Pool.Exec(ctx, `
		UPDATE jobs 
		SET status = 'processing', 
		    updated_at = NOW(),
		    visibility_timeout = $2,
		    worker_id = $3
		WHERE id = $1 AND (
		    status = 'queued' OR 
		    (status = 'processing' AND visibility_timeout < NOW())
		)
	`, id, visibilityTimeout, workerID)

	if err != nil {
		return false, err
	}

	rowsAffected := result.RowsAffected()
	return rowsAffected == 1, nil
}

// CheckRecentDuplicateJob looks for duplicate jobs within a time window
func CheckRecentDuplicateJob(ctx context.Context, contentHash string, windowMinutes int) (*string, error) {
	var existingJobID string
	query := fmt.Sprintf(`
		SELECT id FROM jobs
		WHERE content_hash = $1
		AND created_at > NOW() - INTERVAL '%d minutes'
		AND status IN ('queued', 'processing', 'completed')
		ORDER BY created_at DESC
		LIMIT 1
	`, windowMinutes)

	err := Pool.QueryRow(ctx, query, contentHash).Scan(&existingJobID)

	if err == pgx.ErrNoRows {
		return nil, nil // No recent duplicate
	}
	return &existingJobID, err
}

// ReclaimTimedOutJobs finds jobs that have exceeded their visibility timeout and resets them to 'queued'
// This allows other workers to pick up jobs that were abandoned due to worker crashes
func ReclaimTimedOutJobs(ctx context.Context) (int, error) {
	result, err := Pool.Exec(ctx, `
		UPDATE jobs
		SET status = 'queued',
		    visibility_timeout = NULL,
		    worker_id = NULL,
		    updated_at = NOW()
		WHERE status = 'processing' 
		AND visibility_timeout IS NOT NULL 
		AND visibility_timeout < NOW()
	`)

	if err != nil {
		return 0, err
	}

	return int(result.RowsAffected()), nil
}

// ExtendVisibilityTimeout allows a worker to extend the visibility timeout for a job it's processing
// This prevents the job from being reclaimed while the worker is still actively processing it
func ExtendVisibilityTimeout(ctx context.Context, jobID string, workerID string, additionalMinutes int) error {
	newTimeout := time.Now().Add(time.Duration(additionalMinutes) * time.Minute)

	result, err := Pool.Exec(ctx, `
		UPDATE jobs
		SET visibility_timeout = $1,
		    updated_at = NOW()
		WHERE id = $2 
		AND worker_id = $3 
		AND status = 'processing'
	`, newTimeout, jobID, workerID)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("job not found or not owned by worker %s", workerID)
	}

	return nil
}
