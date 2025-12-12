package ai

import (
	"context"
	"os"
	"strings"
)

// GenerateAIResponse decides whether to use mock or real AI responses.
// Controlled by USE_MOCK_LLM=true/false.
func GenerateAIResponse(ctx context.Context, query string, notes []string) (string, error) {
	useMock := os.Getenv("USE_MOCK_LLM") == "true"

	if useMock {
		return GenerateMockResponse(query, notes), nil
	}

	// Otherwise call real OpenAI (defined in openai.go)
	return GenerateRealAIResponse(ctx, query, notes)
}

// ------------------------------
// MOCK LLM RESPONSE
// ------------------------------

// GenerateMockResponse produces a simple deterministic answer
// based ONLY on the provided notes. It allows the RAG system to run
// without an OpenAI API key.
func GenerateMockResponse(query string, notes []string) string {
	if len(notes) == 0 {
		return "No relevant notes found for your query."
	}

	var builder strings.Builder

	builder.WriteString("Based on your notes, here is a summarized answer:\n\n")

	builder.WriteString("• ")
	builder.WriteString(notes[0])

	if len(notes) > 1 {
		builder.WriteString("\n• " + notes[1])
	}
	if len(notes) > 2 {
		builder.WriteString("\n• " + notes[2])
	}

	builder.WriteString("\n\n(Generated using mock LLM mode)")

	return builder.String()
}
