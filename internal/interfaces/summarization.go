package interfaces

import "context"

// SummarizationProvider defines methods for text summarization
type SummarizationProvider interface {
	SummarizeText(ctx context.Context, text string, prompt string, maxTokens int) (string /*summaryFilePath*/, error)
}
