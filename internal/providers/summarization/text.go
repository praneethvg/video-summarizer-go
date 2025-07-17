package summarization

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"video-summarizer-go/internal/config"
)

// TextSummarizationProvider implements interfaces.SummarizationProvider using text analysis
type TextSummarizationProvider struct {
	promptManager *config.PromptManager
}

// NewTextSummarizationProvider creates a new text-based summarization provider
func NewTextSummarizationProvider() *TextSummarizationProvider {
	return &TextSummarizationProvider{}
}

// NewTextSummarizationProviderFromConfig creates a new text-based summarization provider with prompt manager
func NewTextSummarizationProviderFromConfig(cfg *config.AppConfig) (*TextSummarizationProvider, error) {
	return &TextSummarizationProvider{}, nil
}

// SetPromptManager sets the prompt manager for this provider
func (p *TextSummarizationProvider) SetPromptManager(pm *config.PromptManager) {
	p.promptManager = pm
}

// SummarizeText generates a summary based on the provided text and prompt
func (p *TextSummarizationProvider) SummarizeText(text string, prompt string) (string, error) {
	if text == "" {
		return "No content to summarize.", nil
	}

	// Resolve prompt (either ID or direct content)
	resolvedPrompt, err := p.promptManager.ResolvePrompt(prompt)
	if err != nil {
		return "", fmt.Errorf("failed to resolve prompt: %w", err)
	}

	// For text provider, we'll use the prompt ID to determine the summary type
	// If it's a direct prompt content, default to general summary
	summary := ""
	if strings.Contains(resolvedPrompt, "key points") || strings.Contains(resolvedPrompt, "bullet list") {
		summary = p.generateKeyPoints(text)
	} else if strings.Contains(resolvedPrompt, "timeline") || strings.Contains(resolvedPrompt, "chronological") {
		summary = p.generateTimeline(text)
	} else if strings.Contains(resolvedPrompt, "action items") || strings.Contains(resolvedPrompt, "actionable") {
		summary = p.generateActionItems(text)
	} else if strings.Contains(resolvedPrompt, "educational") || strings.Contains(resolvedPrompt, "learning") {
		summary = p.generateEducationalSummary(text)
	} else if strings.Contains(resolvedPrompt, "meeting") || strings.Contains(resolvedPrompt, "decisions") {
		summary = p.generateMeetingSummary(text)
	} else {
		summary = p.generateGeneralSummary(text)
	}

	tmpFile, err := ioutil.TempFile("", "summary-*.txt")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()
	if _, err := tmpFile.WriteString(summary); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}
	return tmpFile.Name(), nil
}

// GetAvailablePrompts returns the list of available summarization prompts
func (p *TextSummarizationProvider) GetAvailablePrompts() []string {
	prompts := p.promptManager.GetAllPrompts()
	result := make([]string, len(prompts))
	for i, prompt := range prompts {
		result[i] = prompt.ID
	}
	return result
}

// cleanText normalizes and cleans the input text
func (p *TextSummarizationProvider) cleanText(text string) string {
	// Remove extra whitespace
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	// Remove common transcription artifacts
	text = regexp.MustCompile(`\([^)]*\)`).ReplaceAllString(text, "")  // Remove parenthetical notes
	text = regexp.MustCompile(`\[[^\]]*\]`).ReplaceAllString(text, "") // Remove bracketed notes

	return text
}

// generateGeneralSummary creates a general summary of the content
func (p *TextSummarizationProvider) generateGeneralSummary(text string) string {
	// Split into sentences
	sentences := p.splitIntoSentences(text)

	if len(sentences) == 0 {
		return "No meaningful content found."
	}

	// For short content, return as-is
	if len(sentences) <= 3 {
		return strings.Join(sentences, " ")
	}

	// Extract key phrases and create a summary
	keyPhrases := p.extractKeyPhrases(text)

	// Create summary
	summary := "Content Summary:\n"

	// Add main topic identification
	if len(keyPhrases) > 0 {
		summary += fmt.Sprintf("• Main topics: %s\n", strings.Join(keyPhrases[:min(3, len(keyPhrases))], ", "))
	}

	// Add content length info
	wordCount := len(strings.Fields(text))
	summary += fmt.Sprintf("• Content length: %d words\n", wordCount)

	// Add key sentences (first and last meaningful sentences)
	if len(sentences) > 1 {
		summary += fmt.Sprintf("• Opening: %s\n", sentences[0])
		if len(sentences) > 2 {
			summary += fmt.Sprintf("• Closing: %s\n", sentences[len(sentences)-1])
		}
	}

	return summary
}

// generateKeyPoints extracts key points from the content
func (p *TextSummarizationProvider) generateKeyPoints(text string) string {
	// Split into sentences
	sentences := p.splitIntoSentences(text)

	if len(sentences) == 0 {
		return "No key points found."
	}

	// Extract key phrases
	keyPhrases := p.extractKeyPhrases(text)

	// Find sentences with key phrases
	keySentences := p.findKeySentences(sentences, keyPhrases)

	summary := "Key Points:\n"

	// Add key phrases
	if len(keyPhrases) > 0 {
		summary += "• Key topics: " + strings.Join(keyPhrases[:min(5, len(keyPhrases))], ", ") + "\n\n"
	}

	// Add key sentences
	for i, sentence := range keySentences {
		if i >= 5 { // Limit to 5 key points
			break
		}
		summary += fmt.Sprintf("%d. %s\n", i+1, sentence)
	}

	return summary
}

// generateTimeline creates a timeline-based summary
func (p *TextSummarizationProvider) generateTimeline(text string) string {
	// Split into sentences
	sentences := p.splitIntoSentences(text)

	if len(sentences) == 0 {
		return "No timeline information found."
	}

	summary := "Content Timeline:\n"

	// For short content, just number the sentences
	if len(sentences) <= 5 {
		for i, sentence := range sentences {
			summary += fmt.Sprintf("%d. %s\n", i+1, sentence)
		}
		return summary
	}

	// For longer content, group by approximate thirds
	third := len(sentences) / 3

	summary += "Beginning:\n"
	for i := 0; i < min(third, len(sentences)); i++ {
		summary += fmt.Sprintf("• %s\n", sentences[i])
	}

	if len(sentences) > third {
		summary += "\nMiddle:\n"
		for i := third; i < min(2*third, len(sentences)); i++ {
			summary += fmt.Sprintf("• %s\n", sentences[i])
		}
	}

	if len(sentences) > 2*third {
		summary += "\nEnd:\n"
		for i := 2 * third; i < len(sentences); i++ {
			summary += fmt.Sprintf("• %s\n", sentences[i])
		}
	}

	return summary
}

// generateActionItems extracts potential action items from the content
func (p *TextSummarizationProvider) generateActionItems(text string) string {
	// Look for action-oriented words and phrases
	actionWords := []string{
		"need to", "should", "must", "will", "going to", "plan to", "intend to",
		"recommend", "suggest", "propose", "consider", "implement", "create",
		"develop", "build", "design", "test", "review", "analyze", "study",
		"investigate", "research", "explore", "examine", "evaluate", "assess",
	}

	// Split into sentences
	sentences := p.splitIntoSentences(text)

	actionItems := []string{}
	for _, sentence := range sentences {
		lowerSentence := strings.ToLower(sentence)
		for _, actionWord := range actionWords {
			if strings.Contains(lowerSentence, actionWord) {
				actionItems = append(actionItems, sentence)
				break
			}
		}
	}

	if len(actionItems) == 0 {
		return "No specific action items identified."
	}

	summary := "Action Items:\n"
	for i, item := range actionItems {
		if i >= 5 { // Limit to 5 action items
			break
		}
		summary += fmt.Sprintf("%d. %s\n", i+1, item)
	}

	return summary
}

// generateEducationalSummary creates an educational summary of the content
func (p *TextSummarizationProvider) generateEducationalSummary(text string) string {
	// Split into sentences
	sentences := p.splitIntoSentences(text)

	if len(sentences) == 0 {
		return "No educational content found."
	}

	// For short content, return as-is
	if len(sentences) <= 3 {
		return strings.Join(sentences, " ")
	}

	// Extract key phrases
	keyPhrases := p.extractKeyPhrases(text)

	// Create summary
	summary := "Educational Summary:\n"

	// Add main topics
	if len(keyPhrases) > 0 {
		summary += fmt.Sprintf("• Main topics: %s\n", strings.Join(keyPhrases[:min(3, len(keyPhrases))], ", "))
	}

	// Add key sentences (first and last meaningful sentences)
	if len(sentences) > 1 {
		summary += fmt.Sprintf("• Opening: %s\n", sentences[0])
		if len(sentences) > 2 {
			summary += fmt.Sprintf("• Closing: %s\n", sentences[len(sentences)-1])
		}
	}

	return summary
}

// generateMeetingSummary creates a meeting summary of the content
func (p *TextSummarizationProvider) generateMeetingSummary(text string) string {
	// Split into sentences
	sentences := p.splitIntoSentences(text)

	if len(sentences) == 0 {
		return "No meeting content found."
	}

	// For short content, return as-is
	if len(sentences) <= 3 {
		return strings.Join(sentences, " ")
	}

	// Extract key phrases
	keyPhrases := p.extractKeyPhrases(text)

	// Create summary
	summary := "Meeting Summary:\n"

	// Add main topics
	if len(keyPhrases) > 0 {
		summary += fmt.Sprintf("• Main topics: %s\n", strings.Join(keyPhrases[:min(3, len(keyPhrases))], ", "))
	}

	// Add key sentences (first and last meaningful sentences)
	if len(sentences) > 1 {
		summary += fmt.Sprintf("• Opening: %s\n", sentences[0])
		if len(sentences) > 2 {
			summary += fmt.Sprintf("• Closing: %s\n", sentences[len(sentences)-1])
		}
	}

	return summary
}

// splitIntoSentences splits text into sentences
func (p *TextSummarizationProvider) splitIntoSentences(text string) []string {
	// Simple sentence splitting - split on periods, exclamation marks, and question marks
	re := regexp.MustCompile(`[.!?]+`)
	sentences := re.Split(text, -1)

	var result []string
	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if sentence != "" {
			result = append(result, sentence)
		}
	}

	return result
}

// extractKeyPhrases extracts key phrases from the text
func (p *TextSummarizationProvider) extractKeyPhrases(text string) []string {
	// Convert to lowercase for processing
	lowerText := strings.ToLower(text)

	// Remove punctuation
	re := regexp.MustCompile(`[^\w\s]`)
	cleanText := re.ReplaceAllString(lowerText, " ")

	// Split into words
	words := strings.Fields(cleanText)

	// Remove common stop words
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "is": true, "are": true, "was": true, "were": true,
		"be": true, "been": true, "being": true, "have": true, "has": true, "had": true,
		"do": true, "does": true, "did": true, "will": true, "would": true, "could": true,
		"should": true, "may": true, "might": true, "can": true, "this": true, "that": true,
		"these": true, "those": true, "i": true, "you": true, "he": true, "she": true,
		"it": true, "we": true, "they": true, "me": true, "him": true, "her": true,
		"us": true, "them": true, "my": true, "your": true, "his": true,
		"its": true, "our": true, "their": true, "mine": true, "yours": true, "hers": true,
		"ours": true, "theirs": true, "am": true,
		"must": true, "shall": true,
	}

	var filteredWords []string
	for _, word := range words {
		if len(word) > 2 && !stopWords[word] && !unicode.IsDigit(rune(word[0])) {
			filteredWords = append(filteredWords, word)
		}
	}

	// Count word frequencies
	wordFreqMap := make(map[string]int)
	for _, word := range filteredWords {
		wordFreqMap[word]++
	}

	// Sort by frequency
	type wordFreq struct {
		word  string
		count int
	}
	var freqList []wordFreq
	for word, freq := range wordFreqMap {
		freqList = append(freqList, wordFreq{word, freq})
	}

	sort.Slice(freqList, func(i, j int) bool {
		return freqList[i].count > freqList[j].count
	})

	// Return top words as key phrases
	var keyPhrases []string
	for i := 0; i < min(10, len(freqList)); i++ {
		keyPhrases = append(keyPhrases, freqList[i].word)
	}

	return keyPhrases
}

// findKeySentences finds sentences that contain key phrases
func (p *TextSummarizationProvider) findKeySentences(sentences []string, keyPhrases []string) []string {
	var keySentences []string
	for _, sentence := range sentences {
		lowerSentence := strings.ToLower(sentence)
		for _, phrase := range keyPhrases {
			if strings.Contains(lowerSentence, phrase) {
				keySentences = append(keySentences, sentence)
				break
			}
		}
	}
	return keySentences
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
