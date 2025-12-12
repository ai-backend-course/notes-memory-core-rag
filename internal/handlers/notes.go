package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"

	"notes-memory-core-rag/internal/ai"
	"notes-memory-core-rag/internal/database"
)

// Note represents a single note record.
type Note struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// HealthCheck returns a simple service status.
func HealthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "ok",
		"message": "✅ RAG Notes API is healthy and running!",
	})
}

// GetNotes retrieves all notes.
func GetNotes(c *fiber.Ctx) error {
	ctx := context.Background()

	rows, err := database.Pool.Query(ctx,
		`SELECT id, title, content, created_at FROM notes ORDER BY id DESC`)
	if err != nil {
		return c.Status(http.StatusInternalServerError).
			JSON(fiber.Map{"error": err.Error()})
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.Title, &n.Content, &n.CreatedAt); err != nil {
			return c.Status(http.StatusInternalServerError).
				JSON(fiber.Map{"error": err.Error()})
		}
		notes = append(notes, n)
	}

	return c.JSON(notes)
}

// CreateNote inserts a new note and generates an embedding (mock or real).
func CreateNote(c *fiber.Ctx) error {
	ctx := context.Background()

	var n Note
	if err := c.BodyParser(&n); err != nil {
		return c.Status(http.StatusBadRequest).
			JSON(fiber.Map{"error": "invalid request body"})
	}

	if n.Title == "" || n.Content == "" {
		return c.Status(http.StatusBadRequest).
			JSON(fiber.Map{"error": "title and content are required"})
	}

	// Insert the note
	query := `
		INSERT INTO notes (title, content)
		VALUES ($1, $2)
		RETURNING id, created_at
	`
	err := database.Pool.QueryRow(ctx, query, n.Title, n.Content).
		Scan(&n.ID, &n.CreatedAt)
	if err != nil {
		return c.Status(http.StatusInternalServerError).
			JSON(fiber.Map{"error": err.Error()})
	}

	// Generate embedding (mock or real based on env)
	vectorStr, err := ai.GetEmbeddingAsVectorLiteral(ctx, n.Content)
	if err != nil {
		return c.Status(http.StatusInternalServerError).
			JSON(fiber.Map{"error": "failed to generate embedding"})
	}

	// Insert vector
	_, err = database.Pool.Exec(ctx,
		`INSERT INTO note_embeddings (note_id, embedding)
		 VALUES ($1, $2::vector)`,
		n.ID, vectorStr)
	if err != nil {
		return c.Status(http.StatusInternalServerError).
			JSON(fiber.Map{"error": "failed to insert embedding"})
	}

	return c.Status(http.StatusCreated).JSON(n)
}
