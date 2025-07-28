package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// ServiceConfig represents the service configuration
type ServiceConfig struct {
	Server struct {
		Port int    `yaml:"port"`
		Host string `yaml:"host"`
	} `yaml:"server"`

	EngineConfigPath  string `yaml:"engine_config_path"`
	PromptsDir        string `yaml:"prompts_dir"`
	SourcesConfigPath string `yaml:"sources_config_path"`

	// BackgroundSources will be loaded from separate file
	BackgroundSources BackgroundSourcesConfig `yaml:"-"`
}

// BackgroundSourcesConfig represents background sources configuration
type BackgroundSourcesConfig struct {
	Sources []SourceConfig `yaml:"sources"`
}

// SourceConfig represents a background source configuration
type SourceConfig struct {
	Name     string                 `yaml:"name"`
	Type     string                 `yaml:"type"`
	Enabled  bool                   `yaml:"enabled"`
	Interval string                 `yaml:"interval"`
	PromptID string                 `yaml:"prompt_id"`
	Category string                 `yaml:"category"`
	Config   map[string]interface{} `yaml:"config"`
}

func LoadServiceConfig(path string) (*ServiceConfig, error) {
	// Read YAML file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read service config file %s: %w", path, err)
	}

	var cfg ServiceConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse service config file %s: %w", path, err)
	}

	// Apply environment variable overrides
	cfg.applyEnvOverrides()

	// Set defaults for missing values
	cfg.setDefaults()

	// Load background sources from separate file
	if err := cfg.loadBackgroundSources(); err != nil {
		return nil, fmt.Errorf("failed to load background sources: %w", err)
	}

	return &cfg, nil
}

// applyEnvOverrides applies environment variable overrides to the service config
func (c *ServiceConfig) applyEnvOverrides() {
	// Helper function to get env var with fallback
	getEnv := func(key, fallback string) string {
		if val := os.Getenv(key); val != "" {
			return val
		}
		return fallback
	}

	getEnvInt := func(key string, fallback int) int {
		if val := os.Getenv(key); val != "" {
			if i, err := strconv.Atoi(val); err == nil {
				return i
			}
		}
		return fallback
	}

	// Apply server overrides
	c.Server.Port = getEnvInt("VS_SERVER_PORT", c.Server.Port)
	c.Server.Host = getEnv("VS_SERVER_HOST", c.Server.Host)

	// Apply other overrides
	c.EngineConfigPath = getEnv("VS_ENGINE_CONFIG_PATH", c.EngineConfigPath)
	c.PromptsDir = getEnv("VS_PROMPTS_DIR", c.PromptsDir)
	c.SourcesConfigPath = getEnv("VS_SOURCES_CONFIG_PATH", c.SourcesConfigPath)

	// Note: Background sources are configured via YAML config files
	// For runtime configuration, mount different service.yaml files or use ConfigMaps in Kubernetes
}

// setDefaults sets default values for missing service configuration
func (c *ServiceConfig) setDefaults() {
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Server.Host == "" {
		c.Server.Host = "0.0.0.0"
	}
	if c.EngineConfigPath == "" {
		c.EngineConfigPath = "config.yaml"
	}
	if c.PromptsDir == "" {
		c.PromptsDir = "prompts"
	}
	if c.SourcesConfigPath == "" {
		c.SourcesConfigPath = "sources.yaml"
	}
}

// loadBackgroundSources loads background sources from a separate YAML file
func (c *ServiceConfig) loadBackgroundSources() error {
	if c.SourcesConfigPath == "" {
		return nil // No sources config path specified, no sources to load
	}

	data, err := os.ReadFile(c.SourcesConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read sources config file %s: %w", c.SourcesConfigPath, err)
	}

	var sourcesCfg BackgroundSourcesConfig
	if err := yaml.Unmarshal(data, &sourcesCfg); err != nil {
		return fmt.Errorf("failed to parse sources config file %s: %w", c.SourcesConfigPath, err)
	}

	c.BackgroundSources = sourcesCfg
	return nil
}

// GetIntervalDuration returns the parsed interval duration for a source
func (c *SourceConfig) GetIntervalDuration() (time.Duration, error) {
	return time.ParseDuration(c.Interval)
}

// GetMaxVideosPerRun returns the max_videos_per_run value from config
func (c *SourceConfig) GetMaxVideosPerRun() int {
	return c.getConfigInt("max_videos_per_run", 1)
}

// GetChannelVideosLookback returns the channel_videos_lookback value from config
func (c *SourceConfig) GetChannelVideosLookback() int {
	return c.getConfigInt("channel_videos_lookback", 50)
}

// getConfigInt is a reusable helper method to extract integer values from config map
func (c *SourceConfig) getConfigInt(key string, defaultValue int) int {
	if val, ok := c.Config[key]; ok {
		if intVal, ok := val.(int); ok {
			return intVal
		}
		// Handle other numeric types that might be parsed from YAML
		switch v := val.(type) {
		case int64:
			return int(v)
		case float64:
			return int(v)
		}
	}
	return defaultValue
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
