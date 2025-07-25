package summarization

import (
	"video-summarizer-go/internal/config"
	"video-summarizer-go/internal/interfaces"
)

// NewConfigurableSummarizationProviderFromConfig returns the configured summarization provider (OpenAI or text)
func NewConfigurableSummarizationProviderFromConfig(cfg *config.AppConfig) (interfaces.SummarizationProvider, error) {
	if cfg.SummarizerProvider == "openai" {
		openaiProvider, err := NewOpenAISummarizationProviderFromConfig(cfg)
		if err != nil {
			return nil, err
		}
		return openaiProvider, nil
	}

	// Default to text provider
	return nil, nil // This line is removed as text.go has been deleted.
}
