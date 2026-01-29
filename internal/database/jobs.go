package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// Job model
type Job struct {
	ID     string           `json:"id"`
	Type   string           `json:"type"`
	Input  json.RawMessage  `json:"input"`
	Status string           `json:"status"`
	Result *json.RawMessage `json:"result,omitempty"`
	Error  *string          `json:"error,omitempty"`
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
			updated_at = NOW()
		WHERE id = $2 
	`, errMsg, id)
	return err
}

// Fetch job by ID
func GetJobByID(ctx context.Context, id string) (*Job, error) {
	row := Pool.QueryRow(ctx, `
		SELECT id, type, input, status, result, error
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
// Uses PostgreSQL's row-level locking to ensure only one worker can claim each job
// Returns true if successfully claimed, false if already claimed by another worker
func ClaimJobForProcessing(ctx context.Context, id string) (bool, error) {
	result, err := Pool.Exec(ctx, `
		UPDATE jobs 
		SET status = 'processing', updated_at = NOW() 
		WHERE id = $1 AND status = 'queued'
	`, id)

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

	if err == sql.ErrNoRows {
		return nil, nil // No recent duplicate
	}
	return &existingJobID, err
}
