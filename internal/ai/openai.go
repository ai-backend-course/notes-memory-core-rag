package ai

import (
	"context"
	"fmt"
	"os"

	openai "github.com/sashabaranov/go-openai"
)

// GenerateRealAIResponse sends the query + context notes to OpenAI
// and returns a natural-language answer.
func GenerateRealAIResponse(query string, notes []string) (string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("missing OPENAI_API_KEY")
	}

	client := openai.NewClient(apiKey)
	ctx := context.Background()

	// Build RAG-style context block
	contextBlock := "Relevant notes:\n"
	for _, n := range notes {
		contextBlock += "- " + n + "\n"
	}

	// Prompt engineering
	prompt := fmt.Sprintf(`
You are an AI assistant. Use ONLY the provided notes to answer the user's query.

User Query:
%s

%s

Your Answer:
`, query, contextBlock)

	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT4oMini,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
		MaxTokens: 300,
	})
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("empty response from OpenAI")
	}

	return resp.Choices[0].Message.Content, nil
}
