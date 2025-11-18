package ai

import (
	"context"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"math/rand"
	"os"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

// ---------------------------
//  MOCK EMBEDDINGS
// ---------------------------

// GenerateMockEmbedding creates a deterministic embedding (1536-dim)
// based on hashing the input text.
func GenerateMockEmbedding(text string) []float32 {
	const dim = 1536

	hash := sha1.Sum([]byte(text))
	seed := int64(binary.BigEndian.Uint64(hash[:8]))
	r := rand.New(rand.NewSource(seed))

	embedding := make([]float32, dim)
	for i := 0; i < dim; i++ {
		embedding[i] = r.Float32()
	}
	return embedding
}

// ---------------------------
//  REAL OPENAI EMBEDDINGS
// ---------------------------

// GenerateEmbedding calls OpenAI's embeddings API.
func GenerateEmbedding(text string) ([]float32, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("missing OPENAI_API_KEY")
	}

	client := openai.NewClient(apiKey)
	ctx := context.Background()

	resp, err := client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Model: openai.SmallEmbedding3,
		Input: []string{text},
	})
	if err != nil {
		return nil, err
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	return resp.Data[0].Embedding, nil
}

// ---------------------------
//  SELECTION + VECTOR FORMATTER
// ---------------------------

// GetEmbeddingAsVectorLiteral returns a PGVector literal string.
func GetEmbeddingAsVectorLiteral(text string) (string, error) {
	useMock := os.Getenv("USE_MOCK_EMBEDDINGS") == "true"

	// Select mock or real embeddings
	var vec []float32
	var err error

	if useMock {
		vec = GenerateMockEmbedding(text)
	} else {
		vec, err = GenerateEmbedding(text)
		if err != nil {
			return "", err
		}
	}

	// Convert slice → "[0.1,0.2,0.3]"
	builder := strings.Builder{}
	builder.WriteString("[")
	for i, v := range vec {
		if i > 0 {
			builder.WriteString(",")
		}
		builder.WriteString(fmt.Sprintf("%f", v))
	}
	builder.WriteString("]")

	return builder.String(), nil
}
