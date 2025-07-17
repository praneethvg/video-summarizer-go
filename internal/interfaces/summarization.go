package interfaces

// SummarizationProvider defines methods for text summarization
type SummarizationProvider interface {
	SummarizeText(text string, prompt string) (string /*summaryFilePath*/, error)
	GetAvailablePrompts() []string
}
