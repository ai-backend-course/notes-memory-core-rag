package handlers

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"

	"notes-memory-core-rag/internal/database"
)

func GetJob(c *fiber.Ctx) error {
	jobID := c.Params("id")
	if jobID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "job id is required",
		})
	}

	job, err := database.GetJobByID(context.Background(), jobID)
	if err != nil {
		//  This is the ONLY case that should return 404
		if err == sql.ErrNoRows {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{
				"error": "job not found",
			})
		}

		//  All other errors are real server errors
		log.Error().Err(err).Msg("failed to fetch job")

		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to fetch job",
		})
	}

	return c.JSON(job)
}
