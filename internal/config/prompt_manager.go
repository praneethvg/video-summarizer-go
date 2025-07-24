package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// PromptManager manages loading and accessing prompts from files
type PromptManager struct {
	prompts map[string]*Prompt
	loaded  bool
}

// NewPromptManager creates a new prompt manager
func NewPromptManager() *PromptManager {
	return &PromptManager{
		prompts: make(map[string]*Prompt),
		loaded:  false,
	}
}

// LoadPrompts loads all prompt files from the specified directory
func (pm *PromptManager) LoadPrompts(promptsDir string) error {
	if pm.loaded {
		return nil
	}

	// Create prompts directory if it doesn't exist
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		return fmt.Errorf("failed to create prompts directory: %w", err)
	}

	// Load all .yaml files in the prompts directory
	files, err := filepath.Glob(filepath.Join(promptsDir, "*.yaml"))
	if err != nil {
		return fmt.Errorf("failed to glob prompt files: %w", err)
	}

	// If no files exist, create default prompts
	if len(files) == 0 {
		if err := pm.createDefaultPrompts(promptsDir); err != nil {
			return fmt.Errorf("failed to create default prompts: %w", err)
		}
		// Reload after creating defaults
		files, err = filepath.Glob(filepath.Join(promptsDir, "*.yaml"))
		if err != nil {
			return fmt.Errorf("failed to glob prompt files after creating defaults: %w", err)
		}
	}

	// Load each prompt file
	for _, file := range files {
		if err := pm.loadPromptFile(file); err != nil {
			return fmt.Errorf("failed to load prompt file %s: %w", file, err)
		}
	}

	pm.loaded = true
	return nil
}

// loadPromptFile loads a single prompt file
func (pm *PromptManager) loadPromptFile(filepath string) error {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return err
	}

	var prompt Prompt
	if err := yaml.Unmarshal(data, &prompt); err != nil {
		return fmt.Errorf("failed to unmarshal prompt file: %w", err)
	}

	// Validate prompt
	if prompt.ID == "" {
		return fmt.Errorf("prompt in %s has no ID", filepath)
	}
	if prompt.Content == "" {
		return fmt.Errorf("prompt %s has no content", prompt.ID)
	}

	pm.prompts[prompt.ID] = &prompt
	return nil
}

// createDefaultPrompts creates default prompt files
func (pm *PromptManager) createDefaultPrompts(promptsDir string) error {
	defaultPrompts := []Prompt{
		{
			ID:          "general",
			Name:        "General Summary",
			Description: "Creates a general, high-level summary of the content",
			Category:    "summary",
			Content:     "You are an expert at summarizing transcripts. Return a concise, high-level summary of the main content.",
		},
		{
			ID:          "key_points",
			Name:        "Key Points",
			Description: "Extracts the most important key points from the content",
			Category:    "extraction",
			Content:     "You are an expert at extracting the most important key points from transcripts. Return a concise bullet list of the main points.",
		},
		{
			ID:          "timeline",
			Name:        "Timeline",
			Description: "Creates a chronological timeline of events or topics",
			Category:    "organization",
			Content:     "You are an expert at creating timelines from transcripts. Return a chronological list of events or topics as they appear.",
		},
		{
			ID:          "action_items",
			Name:        "Action Items",
			Description: "Identifies actionable tasks and recommendations",
			Category:    "action",
			Content:     "You are an expert at identifying action items from transcripts. Return a bullet list of actionable tasks or recommendations.",
		},
		{
			ID:          "educational",
			Name:        "Educational Summary",
			Description: "Summarizes educational content with learning objectives",
			Category:    "education",
			Content:     "You are an expert at summarizing educational content. Focus on learning objectives, key concepts, and takeaways. Structure the summary to help learners understand the main points.",
		},
		{
			ID:          "meeting",
			Name:        "Meeting Summary",
			Description: "Summarizes meeting content with decisions and next steps",
			Category:    "meeting",
			Content:     "You are an expert at summarizing meetings. Focus on decisions made, action items assigned, and key discussion points. Include who is responsible for what and any deadlines mentioned.",
		},
	}

	for _, prompt := range defaultPrompts {
		if err := pm.savePromptToFile(promptsDir, prompt); err != nil {
			return fmt.Errorf("failed to save default prompt %s: %w", prompt.ID, err)
		}
	}

	return nil
}

// savePromptToFile saves a prompt to a YAML file
func (pm *PromptManager) savePromptToFile(promptsDir string, prompt Prompt) error {
	data, err := yaml.Marshal(prompt)
	if err != nil {
		return err
	}

	filename := filepath.Join(promptsDir, prompt.ID+".yaml")
	return ioutil.WriteFile(filename, data, 0644)
}

// GetPrompt retrieves a prompt by ID
func (pm *PromptManager) GetPrompt(id string) (*Prompt, error) {
	if !pm.loaded {
		return nil, fmt.Errorf("prompts not loaded")
	}

	prompt, exists := pm.prompts[id]
	if !exists {
		return nil, fmt.Errorf("prompt not found: %s", id)
	}

	return prompt, nil
}

// GetPromptContent retrieves the content of a prompt by ID
func (pm *PromptManager) GetPromptContent(id string) (string, error) {
	prompt, err := pm.GetPrompt(id)
	if err != nil {
		return "", err
	}
	return prompt.Content, nil
}

// GetAllPrompts returns all loaded prompts
func (pm *PromptManager) GetAllPrompts() []*Prompt {
	if !pm.loaded {
		return nil
	}

	prompts := make([]*Prompt, 0, len(pm.prompts))
	for _, prompt := range pm.prompts {
		prompts = append(prompts, prompt)
	}
	return prompts
}

// GetPromptsByCategory returns prompts filtered by category
func (pm *PromptManager) GetPromptsByCategory(category string) []*Prompt {
	if !pm.loaded {
		return nil
	}

	var prompts []*Prompt
	for _, prompt := range pm.prompts {
		if strings.EqualFold(prompt.Category, category) {
			prompts = append(prompts, prompt)
		}
	}
	return prompts
}

// ResolvePrompt resolves a prompt input (either ID or direct content)
func (pm *PromptManager) ResolvePrompt(input string) (string, error) {
	if !pm.loaded {
		return "", fmt.Errorf("prompts not loaded")
	}

	// If input looks like a prompt ID (no spaces, alphanumeric + underscore)
	if !strings.Contains(input, " ") && (strings.Contains(input, "_") || len(input) <= 20) {
		// Try to get it as a prompt ID
		if content, err := pm.GetPromptContent(input); err == nil {
			return content, nil
		}
	}

	// If not found as ID or contains spaces, treat as direct content
	return input, nil
}
