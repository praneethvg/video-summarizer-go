package summarization

import (
	"context"
	"fmt"
	"strings"

	"video-summarizer-go/internal/config"

	"io/ioutil"
	"os"

	log "github.com/sirupsen/logrus"

	openai "github.com/sashabaranov/go-openai"
)

// OpenAISummarizationProvider implements interfaces.SummarizationProvider using OpenAI Chat API
type OpenAISummarizationProvider struct {
	client        *openai.Client
	model         string
	maxTokens     int
	promptManager *config.PromptManager
}

func NewOpenAISummarizationProviderFromConfig(cfg *config.AppConfig) (*OpenAISummarizationProvider, error) {
	if cfg.OpenAIKey == "" {
		return nil, fmt.Errorf("openai_api_key not set in config")
	}
	model := cfg.OpenAIModel
	if model == "" {
		model = "gpt-4o"
	}
	maxTokens := cfg.OpenAIMaxTokens
	if maxTokens == 0 {
		maxTokens = 10000 // default
	}
	client := openai.NewClient(cfg.OpenAIKey)

	log.Infof("Initializing provider with model: %s (from config: %s)", model, cfg.OpenAIModel)

	return &OpenAISummarizationProvider{
		client:    client,
		model:     model,
		maxTokens: maxTokens,
	}, nil
}

// SetPromptManager sets the prompt manager for this provider
func (p *OpenAISummarizationProvider) SetPromptManager(pm *config.PromptManager) {
	p.promptManager = pm
}

// SummarizeText summarizes the given text using OpenAI
func (p *OpenAISummarizationProvider) SummarizeText(ctx context.Context, text, prompt string, maxTokens int) (string, error) {
	resolvedPrompt, err := p.promptManager.ResolvePrompt(prompt)
	if err != nil {
		return "", fmt.Errorf("failed to resolve prompt: %w", err)
	}

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: resolvedPrompt,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: text,
		},
	}
	req := openai.ChatCompletionRequest{
		Model:       p.model,
		Messages:    messages,
		MaxTokens:   p.maxTokens,
		Temperature: 0.4,
	}

	log.Debugf("Sending request with model: %s", req.Model)

	resp, err := p.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("OpenAI API error: %w", err)
	}

	log.Debugf("Response received with model: %s", resp.Model)

	summary := strings.TrimSpace(resp.Choices[0].Message.Content)

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

func (p *OpenAISummarizationProvider) GetAvailablePrompts() []string {
	prompts := p.promptManager.GetAllPrompts()
	result := make([]string, len(prompts))
	for i, prompt := range prompts {
		result[i] = prompt.ID
	}
	return result
}
