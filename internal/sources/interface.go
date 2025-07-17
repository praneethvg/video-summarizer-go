package sources

import (
	"context"
	"time"
)

// VideoSource defines the interface for background video sources
type VideoSource interface {
	// Start begins the video source processing
	Start(ctx context.Context) error

	// Stop gracefully stops the video source
	Stop() error

	// GetName returns the name of this video source
	GetName() string

	// IsRunning returns true if the source is currently running
	IsRunning() bool
}

// VideoSourceConfig contains configuration for video sources
type VideoSourceConfig struct {
	Enabled   bool                   `json:"enabled"`
	Interval  time.Duration          `json:"interval"`
	MaxVideos int                    `json:"max_videos_per_run"`
	Queries   []string               `json:"queries,omitempty"`
	Playlists []string               `json:"playlists,omitempty"`
	Channels  []string               `json:"channels,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// VideoSourceManager manages multiple video sources
type VideoSourceManager struct {
	sources map[string]VideoSource
	configs map[string]VideoSourceConfig
}

// NewVideoSourceManager creates a new video source manager
func NewVideoSourceManager() *VideoSourceManager {
	return &VideoSourceManager{
		sources: make(map[string]VideoSource),
		configs: make(map[string]VideoSourceConfig),
	}
}

// AddSource adds a video source to the manager
func (m *VideoSourceManager) AddSource(name string, source VideoSource, config VideoSourceConfig) {
	m.sources[name] = source
	m.configs[name] = config
}

// StartAll starts all enabled video sources
func (m *VideoSourceManager) StartAll(ctx context.Context) error {
	for name, source := range m.sources {
		config := m.configs[name]
		if !config.Enabled {
			continue
		}

		if err := source.Start(ctx); err != nil {
			return err
		}
	}
	return nil
}

// StopAll stops all video sources
func (m *VideoSourceManager) StopAll() error {
	for _, source := range m.sources {
		if source.IsRunning() {
			if err := source.Stop(); err != nil {
				return err
			}
		}
	}
	return nil
}

// GetSource returns a video source by name
func (m *VideoSourceManager) GetSource(name string) (VideoSource, bool) {
	source, exists := m.sources[name]
	return source, exists
}

// GetConfig returns the configuration for a video source
func (m *VideoSourceManager) GetConfig(name string) (VideoSourceConfig, bool) {
	config, exists := m.configs[name]
	return config, exists
}

// GetEnabledSourceNames returns a list of enabled source names
func (m *VideoSourceManager) GetEnabledSourceNames() []string {
	var enabledSources []string
	for name, config := range m.configs {
		if config.Enabled {
			enabledSources = append(enabledSources, name)
		}
	}
	return enabledSources
}
