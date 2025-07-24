package sources

import (
	"context"
	"video-summarizer-go/internal/config"
)

// ArtifactSourceManager manages multiple artifact sources
type ArtifactSourceManager struct {
	sources map[string]ArtifactSource
	configs map[string]*config.SourceConfig
}

// NewArtifactSourceManager creates a new artifact source manager
func NewArtifactSourceManager() *ArtifactSourceManager {
	return &ArtifactSourceManager{
		sources: make(map[string]ArtifactSource),
		configs: make(map[string]*config.SourceConfig),
	}
}

// AddSource adds a video source to the manager
func (m *ArtifactSourceManager) AddSource(name string, source ArtifactSource, config *config.SourceConfig) {
	m.sources[name] = source
	m.configs[name] = config
}

// StartAll starts all enabled video sources
func (m *ArtifactSourceManager) StartAll(ctx context.Context) error {
	for name, source := range m.sources {
		config := m.configs[name]
		if config.Enabled {
			if err := source.Start(ctx); err != nil {
				return err
			}
		}
	}
	return nil
}

// StopAll stops all video sources
func (m *ArtifactSourceManager) StopAll() error {
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
func (m *ArtifactSourceManager) GetSource(name string) (ArtifactSource, bool) {
	source, exists := m.sources[name]
	return source, exists
}

// GetConfig returns the configuration for a video source
func (m *ArtifactSourceManager) GetConfig(name string) (*config.SourceConfig, bool) {
	config, exists := m.configs[name]
	return config, exists
}

// GetEnabledSourceNames returns a list of enabled source names
func (m *ArtifactSourceManager) GetEnabledSourceNames() []string {
	var enabledSources []string
	for name, config := range m.configs {
		if config.Enabled {
			enabledSources = append(enabledSources, name)
		}
	}
	return enabledSources
}
