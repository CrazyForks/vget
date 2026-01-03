package ai

// Model represents an AI model for summarization.
type Model struct {
	ID          string // API model ID
	Name        string // Display name
	Description string // Brief description
	Tier        string // "flagship", "standard", "fast", "economy"
}

// OpenAIModels lists models suitable for text summarization.
// Excludes: image, audio, video, embedding, moderation, codex, transcription models.
// Updated: December 2025
var OpenAIModels = []Model{
	// Flagship models (best quality)
	{ID: "gpt-5.2", Name: "GPT-5.2", Description: "Latest and most capable model", Tier: "flagship"},
	{ID: "gpt-5.2-pro", Name: "GPT-5.2 Pro", Description: "Smarter, more precise responses", Tier: "flagship"},
	{ID: "gpt-5.1", Name: "GPT-5.1", Description: "Excellent for complex tasks", Tier: "flagship"},
	{ID: "gpt-5-pro", Name: "GPT-5 Pro", Description: "Enhanced GPT-5 responses", Tier: "flagship"},
	{ID: "gpt-5", Name: "GPT-5", Description: "Previous flagship model", Tier: "flagship"},

	// Fast models (speed optimized)
	{ID: "gpt-5-mini", Name: "GPT-5 Mini", Description: "Faster GPT-5 for defined tasks", Tier: "fast"},
	{ID: "gpt-4.1-mini", Name: "GPT-4.1 Mini", Description: "Faster version of GPT-4.1", Tier: "fast"},
	{ID: "gpt-4o-mini", Name: "GPT-4o Mini", Description: "Fast, affordable for focused tasks", Tier: "fast"},

	// Economy models (cost optimized)
	{ID: "gpt-5-nano", Name: "GPT-5 Nano", Description: "Most cost-efficient GPT-5", Tier: "economy"},
	{ID: "gpt-4.1-nano", Name: "GPT-4.1 Nano", Description: "Most cost-efficient GPT-4.1", Tier: "economy"},
}

 



// Anthropic models for summarization (January 2025)
var AnthropicModels = []Model{
	{ID: "claude-sonnet-4-5", Name: "Claude Sonnet 4.5", Description: "Latest balanced model", Tier: "flagship"},
	{ID: "claude-haiku-4-5", Name: "Claude Haiku 4.5", Description: "Fast and capable", Tier: "standard"},
	{ID: "claude-opus-4-5", Name: "Claude Opus 4.5", Description: "Most capable", Tier: "flagship"},
 
}


// Default models for each provider
const (
	DefaultOpenAIModel    = "gpt-5.2-pro"
	DefaultAnthropicModel = "claude-haiku-4-5"
)

// GetModelByID returns model info by ID from any provider, or nil if not found.
func GetModelByID(id string) *Model {
	for _, m := range OpenAIModels {
		if m.ID == id {
			return &m
		}
	}
	for _, m := range AnthropicModels {
		if m.ID == id {
			return &m
		}
	}

	return nil
}
