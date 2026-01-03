package summarizer

import (
	"context"
	"fmt"

	"github.com/guiyumin/vget/internal/core/config"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// ProviderEndpoints maps provider names to their OpenAI-compatible API endpoints.
var ProviderEndpoints = map[string]string{
	"deepseek":   "https://api.deepseek.com/v1",
	"moonshot":   "https://api.moonshot.cn/v1",
	"zhipu":      "https://open.bigmodel.cn/api/paas/v4",
	"minimax":    "https://api.minimax.chat/v1",
	"baichuan":   "https://api.baichuan-ai.com/v1",
	"volcengine": "https://ark.cn-beijing.volces.com/api/v3",
}

// OpenAICompat implements Summarizer using OpenAI-compatible APIs.
type OpenAICompat struct {
	client   openai.Client
	model    string
	provider string
}

// NewOpenAICompat creates a new OpenAI-compatible summarizer.
func NewOpenAICompat(provider string, cfg config.AIServiceConfig, apiKey string) (*OpenAICompat, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("%s API key not provided", provider)
	}

	// Get base URL from config or use default for provider
	baseURL := cfg.BaseURL
	if baseURL == "" {
		var ok bool
		baseURL, ok = ProviderEndpoints[provider]
		if !ok {
			return nil, fmt.Errorf("unknown provider: %s", provider)
		}
	}

	opts := []option.RequestOption{
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseURL),
	}

	client := openai.NewClient(opts...)

	// Model is required - no defaults
	if cfg.Model == "" {
		return nil, fmt.Errorf("%s model not specified", provider)
	}

	return &OpenAICompat{
		client:   client,
		model:    cfg.Model,
		provider: provider,
	}, nil
}

// Name returns the provider name.
func (o *OpenAICompat) Name() string {
	return o.provider
}

// Summarize generates a summary from the given text.
func (o *OpenAICompat) Summarize(ctx context.Context, text string) (*Result, error) {
	// Truncate text if too long
	maxChars := 100000
	if len(text) > maxChars {
		text = text[:maxChars] + "\n\n[Text truncated due to length...]"
	}

	resp, err := o.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.ChatModel(o.model),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(SummarizationPrompt + text),
		},
		MaxTokens:   openai.Int(8000),
		Temperature: openai.Float(0.3),
	})
	if err != nil {
		return nil, fmt.Errorf("summarization API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from API")
	}

	return parseResponse(resp.Choices[0].Message.Content), nil
}

// Translate translates the text to the target language.
func (o *OpenAICompat) Translate(ctx context.Context, text string, targetLang string) (string, error) {
	maxChars := 100000
	if len(text) > maxChars {
		text = text[:maxChars] + "\n\n[Text truncated due to length...]"
	}

	prompt := fmt.Sprintf("Translate the following text to %s. Preserve the original formatting, structure, and any timestamps. Only output the translated text, no explanations.\n\n%s", targetLang, text)

	resp, err := o.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.ChatModel(o.model),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
		MaxTokens:   openai.Int(8000),
		Temperature: openai.Float(0.3),
	})
	if err != nil {
		return "", fmt.Errorf("translation API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	return resp.Choices[0].Message.Content, nil
}
