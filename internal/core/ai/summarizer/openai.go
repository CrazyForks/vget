package summarizer

import (
	"context"
	"fmt"
	"strings"

	"github.com/guiyumin/vget/internal/core/config"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// defaultOpenAIModel is the default model for summarization.
const defaultOpenAIModel = "gpt-5-nano"

// OpenAI implements Summarizer using OpenAI GPT (official SDK).
type OpenAI struct {
	client openai.Client
	model  openai.ChatModel
}

// NewOpenAI creates a new OpenAI summarizer.
// The apiKey parameter is the decrypted API key.
func NewOpenAI(cfg config.AIServiceConfig, apiKey string) (*OpenAI, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key not provided")
	}

	opts := []option.RequestOption{
		option.WithAPIKey(apiKey),
	}
	if cfg.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.BaseURL))
	}

	client := openai.NewClient(opts...)

	model := openai.ChatModel(cfg.Model)
	if cfg.Model == "" {
		model = openai.ChatModel(defaultOpenAIModel)
	}

	return &OpenAI{
		client: client,
		model:  model,
	}, nil
}

// Name returns the provider name.
func (o *OpenAI) Name() string {
	return "openai"
}

// Summarize generates a summary from the given text using OpenAI GPT.
func (o *OpenAI) Summarize(ctx context.Context, text string) (*Result, error) {
	// Truncate text if too long (GPT-4o has 128k context but we want to be efficient)
	maxChars := 100000
	if len(text) > maxChars {
		text = text[:maxChars] + "\n\n[Text truncated due to length...]"
	}

	// Create chat completion request
	// Note: Avoid MaxTokens/MaxCompletionTokens/Temperature as newer models (o1, gpt-5) don't support them
	resp, err := o.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: o.model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(SummarizationPrompt + text),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("summarization API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from API")
	}

	content := resp.Choices[0].Message.Content

	// Parse response
	return parseResponse(content), nil
}

// parseResponse extracts summary and key points from the response.
func parseResponse(content string) *Result {
	trimmed := strings.TrimSpace(content)
	result := &Result{
		Summary: trimmed,
	}

	// Try to extract key points (legacy format) without stripping headings.
	lines := strings.Split(trimmed, "\n")
	var keyPoints []string
	inKeyPoints := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "## Key Points") || strings.HasPrefix(line, "**Key Points") {
			inKeyPoints = true
			continue
		}

		if strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "### ") || strings.HasPrefix(line, "**") {
			if inKeyPoints {
				inKeyPoints = false
			}
		}

		if inKeyPoints && (strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*")) {
			point := strings.TrimPrefix(line, "-")
			point = strings.TrimPrefix(point, "*")
			point = strings.TrimSpace(point)
			if point != "" {
				keyPoints = append(keyPoints, point)
			}
		}
	}

	if len(keyPoints) > 0 {
		result.KeyPoints = keyPoints
	}

	return result
}
