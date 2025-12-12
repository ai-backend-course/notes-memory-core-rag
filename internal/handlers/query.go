package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"notes-memory-core-rag/internal/ai"
	"notes-memory-core-rag/internal/database"
)

// QueryRequest represents a semantic search or RAG request.
type QueryRequest struct {
	Query string `json:"query"`
}

// SemanticSearch performs vector similarity search using pgvector.
func SemanticSearch(c *fiber.Ctx) error {
	var req QueryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if strings.TrimSpace(req.Query) == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "query text is required",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	// Get embedding for query (mock or real)
	queryVec, err := ai.GetEmbeddingAsVectorLiteral(ctx, req.Query)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to generate embedding",
		})
	}

	// Perform similarity search (<-> operator)
	rows, err := database.Pool.Query(ctx, `
		SELECT n.id, n.title, n.content, n.created_at,
		       e.embedding <-> $1::vector AS distance
		FROM notes n
		JOIN note_embeddings e ON n.id = e.note_id
		ORDER BY distance ASC
		LIMIT 5;
	`, queryVec)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	defer rows.Close()

	type SearchResult struct {
		ID        int       `json:"id"`
		Title     string    `json:"title"`
		Content   string    `json:"content"`
		CreatedAt time.Time `json:"created_at"`
		Distance  float64   `json:"distance"`
	}

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(
			&r.ID,
			&r.Title,
			&r.Content,
			&r.CreatedAt,
			&r.Distance,
		); err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		results = append(results, r)
	}

	return c.JSON(fiber.Map{
		"query":   req.Query,
		"results": results,
	})
}

// Query performs full RAG: semantic search + AI-generated answer.
func Query(c *fiber.Ctx) error {
	var req QueryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if strings.TrimSpace(req.Query) == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "query text is required",
		})
	}

	result, err := RunRAGPipeline(context.Background(), req.Query)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(result)
}
