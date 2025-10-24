package llm

import (
	"fmt"
)

// Provider represents an LLM provider
type Provider string

const (
	ProviderGemini Provider = "gemini"
	ProviderClaude Provider = "claude"
	ProviderOpenAI Provider = "openai"
)

// Client interface for LLM providers
type Client interface {
	Analyze(eventType, podName, namespace, logs string) (string, error)
}

// NewClient creates a new LLM client based on provider
func NewClient(provider Provider, apiKey string) (Client, error) {
	switch provider {
	case ProviderGemini:
		return NewGeminiClient(apiKey), nil
	case ProviderClaude:
		return NewClaudeClient(apiKey), nil
	case ProviderOpenAI:
		return NewOpenAIClient(apiKey), nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}
