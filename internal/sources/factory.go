package sources

import (
	"fmt"
	"time"

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
func (f *SourceFactory) CreateSource(sourceConfig *config.SourceConfig, ytDlpPath string) (VideoSource, error) {
	if !sourceConfig.Enabled {
		return nil, fmt.Errorf("source %s is disabled", sourceConfig.Name)
	}

	interval, err := sourceConfig.GetIntervalDuration()
	if err != nil {
		return nil, fmt.Errorf("invalid interval for source %s: %w", sourceConfig.Name, err)
	}

	// Set default prompt if not specified
	promptID := sourceConfig.PromptID
	if promptID == "" {
		promptID = "general"
	}

	// Create metadata for this source
	metadata := map[string]interface{}{
		"auto_discovered": true,
		"source_name":     sourceConfig.Name,
		"source_type":     sourceConfig.Type,
		"prompt":          promptID,
	}

	switch sourceConfig.Type {
	case "youtube_search":
		return f.createYouTubeSearchSource(sourceConfig, interval, ytDlpPath)
	case "rss_feed":
		return f.createRSSFeedSource(sourceConfig, interval, metadata)
	default:
		return nil, fmt.Errorf("unsupported source type: %s", sourceConfig.Type)
	}
}

// createYouTubeSearchSource creates a YouTube search source
func (f *SourceFactory) createYouTubeSearchSource(sourceConfig *config.SourceConfig, interval time.Duration, ytDlpPath string) (VideoSource, error) {
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

	// Set default channel videos lookback if not specified
	channelVideosLookback := sourceConfig.ChannelVideosLookback
	if channelVideosLookback == 0 {
		channelVideosLookback = 50 // Default to scanning 50 videos
	}

	category := "general"
	if sourceConfig.Category != "" {
		category = sourceConfig.Category
	}
	return NewSearchQuerySource(
		sourceConfig.Name,
		queries,
		channel,
		interval,
		sourceConfig.MaxVideosPerRun,
		channelVideosLookback,
		ytDlpPath,
		f.submissionService,
		category,
		sourceConfig.PromptID,
	), nil
}

// createRSSFeedSource creates an RSS feed source (placeholder for future implementation)
func (f *SourceFactory) createRSSFeedSource(sourceConfig *config.SourceConfig, interval time.Duration, metadata map[string]interface{}) (VideoSource, error) {
	feedURL, err := sourceConfig.GetFeedURL()
	if err != nil {
		return nil, fmt.Errorf("failed to get feed URL for source %s: %w", sourceConfig.Name, err)
	}

	// TODO: Implement RSS feed source
	return nil, fmt.Errorf("RSS feed source not yet implemented for source %s (feed: %s)", sourceConfig.Name, feedURL)
}
