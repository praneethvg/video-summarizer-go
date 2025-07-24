package sources

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"video-summarizer-go/internal/interfaces"
	"video-summarizer-go/internal/services"
)

// SearchQuerySource implements VideoSource for YouTube search queries
type SearchQuerySource struct {
	name                  string
	queries               []string
	channel               string // Only one channel per source
	interval              time.Duration
	maxVideos             int
	channelVideosLookback int // How many videos to scan when searching within a channel
	ytDlpPath             string
	submissionService     *services.VideoSubmissionService
	Category              string
	PromptID              string

	running bool
	stopCh  chan struct{}
	mu      sync.RWMutex
}

// NewSearchQuerySource creates a new search query video source
func NewSearchQuerySource(
	name string,
	queries []string,
	channel string,
	interval time.Duration,
	maxVideos int,
	channelVideosLookback int,
	ytDlpPath string,
	submissionService *services.VideoSubmissionService,
	category string,
	promptID string,
) *SearchQuerySource {
	return &SearchQuerySource{
		name:                  name,
		queries:               queries,
		channel:               channel,
		interval:              interval,
		maxVideos:             maxVideos,
		channelVideosLookback: channelVideosLookback,
		ytDlpPath:             ytDlpPath,
		submissionService:     submissionService,
		Category:              category,
		PromptID:              promptID,
		stopCh:                make(chan struct{}),
	}
}

// Start begins the search query processing
func (s *SearchQuerySource) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("search source %s is already running", s.name)
	}

	s.running = true
	s.stopCh = make(chan struct{})

	go s.run(ctx)

	log.Infof("Started search source: %s", s.name)
	return nil
}

// Stop gracefully stops the search query processing
func (s *SearchQuerySource) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	close(s.stopCh)
	s.running = false

	log.Infof("Stopped search source: %s", s.name)
	return nil
}

// GetName returns the name of this video source
func (s *SearchQuerySource) GetName() string {
	return s.name
}

// IsRunning returns true if the source is currently running
func (s *SearchQuerySource) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// run is the main processing loop
func (s *SearchQuerySource) run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Run immediately on start
	s.processQueries()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.processQueries()
		}
	}
}

// processQueries processes all configured search queries
func (s *SearchQuerySource) processQueries() {
	log.Infof("Processing %d queries for source: %s", len(s.queries), s.name)

	for _, query := range s.queries {
		videos, err := s.searchVideos(query)
		if err != nil {
			log.Errorf("Error searching for query '%s': %v", query, err)
			continue
		}

		if len(videos) == 0 {
			log.Warnf("No videos found for query: %s", query)
			continue
		}

		// Limit the number of videos per query
		if len(videos) > s.maxVideos {
			videos = videos[:s.maxVideos]
		}

		prompt := s.PromptID
		if prompt == "" {
			prompt = "general"
		}
		promptStruct := interfaces.Prompt{Type: interfaces.PromptTypeID, Prompt: prompt}
		sourceType := "video"
		category := s.Category
		if category == "" {
			category = "general"
		}
		maxTokens := 10000
		// Submit videos for processing
		requestIDs, err := s.submissionService.SubmitBatch(videos, promptStruct, sourceType, category, maxTokens)
		if err != nil {
			log.Errorf("Error submitting videos for query '%s': %v", query, err)
			continue
		}

		log.Infof("Submitted %d videos for query '%s': %v", len(requestIDs), query, requestIDs)
	}
}

// searchVideos uses yt-dlp to search for videos
func (s *SearchQuerySource) searchVideos(query string) ([]string, error) {
	log.Debugf("Starting search for query: '%s' (channel: %s)", query, s.channel)

	var shellCmd string

	if s.channel != "" {
		// Use --match-title with channel videos URL when channel is provided
		// Scan through channelVideosLookback videos, then limit to maxVideos results
		channelURL := fmt.Sprintf("https://www.youtube.com/channel/%s/videos", s.channel)
		shellCmd = fmt.Sprintf("%s --match-title '%s' --print '%%(id)s' --flat-playlist --simulate -I :%d %s | head -%d",
			s.ytDlpPath, query, s.channelVideosLookback, channelURL, s.maxVideos)
		log.Debugf("Using channel-specific search with --match-title (scanning %d videos, will return up to %d)", s.channelVideosLookback, s.maxVideos)
	} else {
		// Use ytsearch when no channel is specified
		searchArg := fmt.Sprintf("ytsearch%d:%s", s.maxVideos, strings.TrimSpace(query))
		shellCmd = fmt.Sprintf("%s '%s' --get-id --no-playlist", s.ytDlpPath, searchArg)
		log.Debugf("Using general ytsearch (no channel filter)")
	}

	// Print the full command as it would appear in the shell
	log.Debugf("Full yt-dlp command: %s", shellCmd)

	cmd := exec.Command("sh", "-c", shellCmd)
	output, err := cmd.Output()
	if err != nil {
		log.Errorf("yt-dlp search failed for query '%s': %v", query, err)
		return nil, fmt.Errorf("yt-dlp search failed: %w", err)
	}

	log.Debugf("yt-dlp output for query '%s':\n%s", query, string(output))

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var videoURLs []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		videoURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", line)
		videoURLs = append(videoURLs, videoURL)
		if len(videoURLs) >= s.maxVideos {
			break
		}
	}

	log.Infof("Found %d video(s) for query '%s' (channel: %s)", len(videoURLs), query, s.channel)
	return videoURLs, nil
}
