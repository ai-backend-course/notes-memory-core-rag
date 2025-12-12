package handlers

import (
	"context"
	"notes-memory-core-rag/internal/ai"
	"notes-memory-core-rag/internal/database"
	"time"
)

type SearchResult struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	Distance  float64   `json:"distance"`
}

type RAGResult struct {
	Query    string         `json:"query"`
	Response string         `json:"response"`
	Results  []SearchResult `json:"results"`
}

func RunRAGPipeline(parentCtx context.Context, query string) (*RAGResult, error) {
	// Enforce an upper bound for the entire pipeline
	ctx, cancel := context.WithTimeout(parentCtx, 15*time.Second)
	defer cancel()

	// 1. Embed text
	queryVec, err := ai.GetEmbeddingAsVectorLiteral(query)
	if err != nil {
		return nil, err
	}

	// 2. Vector similarity search
	rows, err := database.Pool.Query(ctx, `
		SELECT n.id, n.title, n.content, n.created_at,
			e.embedding <-> $1::vector AS distance
		FROM notes n
		JOIN note_embeddings e ON n.id = e.note_id
		ORDER BY distance ASC
		LIMIT 3;
		`, queryVec)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	var contextTexts []string

	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.ID, &r.Title, &r.Content, &r.CreatedAt, &r.Distance); err != nil {
			return nil, err
		}

		results = append(results, r)
		contextTexts = append(contextTexts, r.Content)

	}

	// 3. LLM answer generation
	aiResponse, err := ai.GenerateAIResponse(query, contextTexts)
	if err != nil {
		return nil, err
	}

	return &RAGResult{
		Query:    query,
		Response: aiResponse,
		Results:  results,
	}, nil
}
