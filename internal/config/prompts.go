package config

// Prompt represents a single prompt definition
type Prompt struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Content     string `yaml:"content"`
	Category    string `yaml:"category"`
}
