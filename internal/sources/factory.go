package sources

import (
	"fmt"

	"video-summarizer-go/internal/config"
	"video-summarizer-go/internal/services"
)

// SourceFactory creates video sources based on configuration
type SourceFactory struct {
	submissionService *services.VideoSubmissionService
}

// NewSourceFactory creates a new source factory
func NewSourceFactory(submissionService *services.VideoSubmissionService) *SourceFactory {
	return &SourceFactory{
		submissionService: submissionService,
	}
}

// CreateSource creates a video source based on the source configuration
func (f *SourceFactory) CreateSource(sourceConfig *config.SourceConfig, appCfg *config.AppConfig) (ArtifactSource, error) {
	if !sourceConfig.Enabled {
		return nil, fmt.Errorf("source %s is disabled", sourceConfig.Name)
	}

	_, err := sourceConfig.GetIntervalDuration()
	if err != nil {
		return nil, fmt.Errorf("invalid interval for source %s: %w", sourceConfig.Name, err)
	}

	if sourceConfig.PromptID == "" {
		sourceConfig.PromptID = "general"
	}

	switch sourceConfig.Type {
	case "youtube_search":
		return f.createYouTubeSearchSource(sourceConfig, appCfg)
	default:
		return nil, fmt.Errorf("unsupported source type: %s", sourceConfig.Type)
	}
}

// createYouTubeSearchSource creates a YouTube search source
func (f *SourceFactory) createYouTubeSearchSource(sourceConfig *config.SourceConfig, appCfg *config.AppConfig) (ArtifactSource, error) {
	queries, err := sourceConfig.GetQueries()
	if err != nil {
		return nil, fmt.Errorf("failed to get queries for source %s: %w", sourceConfig.Name, err)
	}

	if len(queries) == 0 {
		return nil, fmt.Errorf("no queries specified for source %s", sourceConfig.Name)
	}

	// Get channel from config if specified
	channel := ""
	if ch, ok := sourceConfig.Config["channel"].(string); ok {
		channel = ch
	}

	// Read max_videos_per_run and channel_videos_lookback from config map with defaults
	maxVideosPerRun := 1
	if val, ok := sourceConfig.Config["max_videos_per_run"]; ok {
		maxVideosPerRun = val.(int)
	}

	channelVideosLookback := 50
	if val, ok := sourceConfig.Config["channel_videos_lookback"]; ok {
		channelVideosLookback = val.(int)
	}

	category := "general"
	if sourceConfig.Category != "" {
		category = sourceConfig.Category
	}

	interval, _ := sourceConfig.GetIntervalDuration()
	return NewSearchQuerySource(
		sourceConfig.Name,
		queries,
		channel,
		interval,
		maxVideosPerRun,
		channelVideosLookback,
		appCfg.YtDlpPath,
		f.submissionService,
		category,
		sourceConfig.PromptID,
	), nil
}
