package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// ServiceConfig represents the configuration for the video summarizer service
type ServiceConfig struct {
	Server            ServerConfig            `yaml:"server"`
	EngineConfigPath  string                  `yaml:"engine_config_path"`
	BackgroundSources BackgroundSourcesConfig `yaml:"background_sources"`
	PromptsDir        string                  `yaml:"prompts_dir"`
}

// ServerConfig defines HTTP server settings
type ServerConfig struct {
	Port int    `yaml:"port"`
	Host string `yaml:"host"`
}

// BackgroundSourcesConfig defines background video source settings
type BackgroundSourcesConfig struct {
	Sources []SourceConfig `yaml:"sources"`
}

// SourceConfig represents a single background video source configuration
type SourceConfig struct {
	Name                  string                 `yaml:"name"`
	Type                  string                 `yaml:"type"`
	Enabled               bool                   `yaml:"enabled"`
	Interval              string                 `yaml:"interval"`
	MaxVideosPerRun       int                    `yaml:"max_videos_per_run"`
	ChannelVideosLookback int                    `yaml:"channel_videos_lookback"` // How many videos to scan when searching within a channel
	PromptID              string                 `yaml:"prompt_id"`
	Category              string                 `yaml:"category"`
	Config                map[string]interface{} `yaml:"config"`
}

// LoadServiceConfig loads the service configuration from a file
func LoadServiceConfig(path string) (*ServiceConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open service config file: %w", err)
	}
	defer f.Close()

	var cfg ServiceConfig
	dec := yaml.NewDecoder(f)
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to decode service config: %w", err)
	}

	// Set defaults
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}
	if cfg.EngineConfigPath == "" {
		cfg.EngineConfigPath = "config.yaml"
	}

	return &cfg, nil
}

// GetIntervalDuration parses the interval string and returns a duration
func (c *SourceConfig) GetIntervalDuration() (time.Duration, error) {
	if c.Interval == "" {
		return 30 * time.Minute, nil // default
	}
	return time.ParseDuration(c.Interval)
}

// GetQueries extracts queries from config for youtube_search type
func (c *SourceConfig) GetQueries() ([]string, error) {
	if c.Type != "youtube_search" {
		return nil, fmt.Errorf("queries not supported for source type: %s", c.Type)
	}

	queriesInterface, ok := c.Config["queries"]
	if !ok {
		return nil, fmt.Errorf("queries not found in config for source: %s", c.Name)
	}

	queries, ok := queriesInterface.([]interface{})
	if !ok {
		return nil, fmt.Errorf("queries must be a list for source: %s", c.Name)
	}

	result := make([]string, len(queries))
	for i, q := range queries {
		if str, ok := q.(string); ok {
			result[i] = str
		} else {
			return nil, fmt.Errorf("query must be a string for source: %s", c.Name)
		}
	}

	return result, nil
}

// GetFeedURL extracts feed URL from config for rss_feed type
func (c *SourceConfig) GetFeedURL() (string, error) {
	if c.Type != "rss_feed" {
		return "", fmt.Errorf("feed_url not supported for source type: %s", c.Type)
	}

	feedURLInterface, ok := c.Config["feed_url"]
	if !ok {
		return "", fmt.Errorf("feed_url not found in config for source: %s", c.Name)
	}

	feedURL, ok := feedURLInterface.(string)
	if !ok {
		return "", fmt.Errorf("feed_url must be a string for source: %s", c.Name)
	}

	return feedURL, nil
}
