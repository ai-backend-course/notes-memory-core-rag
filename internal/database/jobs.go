package database

import (
	"context"
	"encoding/json"

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

// Create new job in DB
func CreateJob(ctx context.Context, jobType string, input interface{}) (string, error) {
	id := uuid.New().String()

	// Convert input payload to JSON
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return "", err
	}

	// Insert into database
	_, err = Pool.Exec(ctx, `
		INSERT INTO jobs (id, type, input, status)
		VALUES ($1, $2, $3, 'queued')
	`, id, jobType, inputBytes)

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
