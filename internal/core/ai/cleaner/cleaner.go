// Package cleaner provides post-transcription cleanup using LLMs.
package cleaner

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	anthropicopt "github.com/anthropics/anthropic-sdk-go/option"
	"github.com/guiyumin/vget/internal/core/config"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// Cleaner cleans raw transcripts using an LLM.
type Cleaner interface {
	// Clean removes filler words, fixes punctuation, etc.
	Clean(ctx context.Context, rawText string) (string, error)
}

// New creates a Cleaner based on provider.
func New(provider string, cfg config.AIServiceConfig, apiKey string) (Cleaner, error) {
	switch provider {
	case "openai":
		return newOpenAICleaner(cfg, apiKey)
	case "anthropic":
		return newAnthropicCleaner(cfg, apiKey)
	case "qwen":
		return newQwenCleaner(cfg, apiKey)
	default:
		return nil, fmt.Errorf("unsupported cleaner provider: %s", provider)
	}
}

// openAICleaner uses OpenAI for cleaning.
type openAICleaner struct {
	client openai.Client
	model  openai.ChatModel
}

func newOpenAICleaner(cfg config.AIServiceConfig, apiKey string) (*openAICleaner, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key not provided")
	}

	opts := []option.RequestOption{
		option.WithAPIKey(apiKey),
	}
	if cfg.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.BaseURL))
	}

	model := openai.ChatModel(cfg.Model)
	if cfg.Model == "" {
		model = openai.ChatModelGPT4oMini // Fast and cheap for cleaning
	}

	return &openAICleaner{
		client: openai.NewClient(opts...),
		model:  model,
	}, nil
}

func (c *openAICleaner) Clean(ctx context.Context, rawText string) (string, error) {
	resp, err := c.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: c.model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(CleaningPrompt + rawText),
		},
		Temperature: openai.Float(0.1), // Low temperature for consistent cleaning
	})
	if err != nil {
		return "", fmt.Errorf("cleaning API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	return resp.Choices[0].Message.Content, nil
}

// anthropicCleaner uses Anthropic Claude for cleaning.
type anthropicCleaner struct {
	client *anthropic.Client
	model  string
}

func newAnthropicCleaner(cfg config.AIServiceConfig, apiKey string) (*anthropicCleaner, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Anthropic API key not provided")
	}

	opts := []anthropicopt.RequestOption{
		anthropicopt.WithAPIKey(apiKey),
	}
	if cfg.BaseURL != "" {
		opts = append(opts, anthropicopt.WithBaseURL(cfg.BaseURL))
	}

	client := anthropic.NewClient(opts...)

	model := cfg.Model
	if model == "" {
		model = "claude-3-5-haiku-latest" // Fast and cheap for cleaning
	}

	return &anthropicCleaner{
		client: &client,
		model:  model,
	}, nil
}

func (c *anthropicCleaner) Clean(ctx context.Context, rawText string) (string, error) {
	message, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(c.model),
		MaxTokens: 8000,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(CleaningPrompt + rawText)),
		},
	})
	if err != nil {
		return "", fmt.Errorf("cleaning API error: %w", err)
	}

	var content string
	for _, block := range message.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	if content == "" {
		return "", fmt.Errorf("no response from API")
	}

	return content, nil
}

// qwenCleaner uses Alibaba Qwen for cleaning.
type qwenCleaner struct {
	client openai.Client
	model  string
}

const qwenDefaultBaseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"

func newQwenCleaner(cfg config.AIServiceConfig, apiKey string) (*qwenCleaner, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Qwen API key not provided")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = qwenDefaultBaseURL
	}

	opts := []option.RequestOption{
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseURL),
	}

	model := cfg.Model
	if model == "" {
		model = "qwen-turbo" // Fast and cheap for cleaning
	}

	return &qwenCleaner{
		client: openai.NewClient(opts...),
		model:  model,
	}, nil
}

func (c *qwenCleaner) Clean(ctx context.Context, rawText string) (string, error) {
	resp, err := c.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.ChatModel(c.model),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(CleaningPrompt + rawText),
		},
		Temperature: openai.Float(0.1),
	})
	if err != nil {
		return "", fmt.Errorf("cleaning API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	return resp.Choices[0].Message.Content, nil
}
