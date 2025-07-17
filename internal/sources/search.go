package sources

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"

	"video-summarizer-go/internal/services"
)

// SearchQuerySource implements VideoSource for YouTube search queries
type SearchQuerySource struct {
	name              string
	queries           []string
	channels          []string // Channel IDs or names to filter by
	interval          time.Duration
	maxVideos         int
	ytDlpPath         string
	submissionService *services.VideoSubmissionService
	metadata          map[string]interface{}

	running bool
	stopCh  chan struct{}
	mu      sync.RWMutex
}

// NewSearchQuerySource creates a new search query video source
func NewSearchQuerySource(
	name string,
	queries []string,
	channels []string,
	interval time.Duration,
	maxVideos int,
	ytDlpPath string,
	submissionService *services.VideoSubmissionService,
	metadata map[string]interface{},
) *SearchQuerySource {
	return &SearchQuerySource{
		name:              name,
		queries:           queries,
		channels:          channels,
		interval:          interval,
		maxVideos:         maxVideos,
		ytDlpPath:         ytDlpPath,
		submissionService: submissionService,
		metadata:          metadata,
		stopCh:            make(chan struct{}),
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

	log.Printf("[SearchSource] Started search source: %s", s.name)
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

	log.Printf("[SearchSource] Stopped search source: %s", s.name)
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
	log.Printf("[SearchSource] Processing %d queries for source: %s", len(s.queries), s.name)

	for _, query := range s.queries {
		videos, err := s.searchVideos(query)
		if err != nil {
			log.Printf("[SearchSource] Error searching for query '%s': %v", query, err)
			continue
		}

		if len(videos) == 0 {
			log.Printf("[SearchSource] No videos found for query: %s", query)
			continue
		}

		// Limit the number of videos per query
		if len(videos) > s.maxVideos {
			videos = videos[:s.maxVideos]
		}

		// Add metadata for this source
		metadata := make(map[string]interface{})
		for k, v := range s.metadata {
			metadata[k] = v
		}
		metadata["source"] = "search_query"
		metadata["query"] = query
		metadata["source_name"] = s.name

		prompt := "general"
		if p, ok := metadata["prompt"].(string); ok && p != "" {
			prompt = p
		}
		// Submit videos for processing
		requestIDs, err := s.submissionService.SubmitBatch(videos, prompt, metadata)
		if err != nil {
			log.Printf("[SearchSource] Error submitting videos for query '%s': %v", query, err)
			continue
		}

		log.Printf("[SearchSource] Submitted %d videos for query '%s': %v", len(requestIDs), query, requestIDs)
	}
}

// searchVideos uses yt-dlp to search for videos
func (s *SearchQuerySource) searchVideos(query string) ([]string, error) {
	// Use yt-dlp to search for videos
	// yt-dlp "ytsearch10:query" --get-id --no-playlist
	searchArg := fmt.Sprintf("ytsearch%d:%s", s.maxVideos*2, query) // Search more to account for filtering

	cmd := exec.Command(s.ytDlpPath, searchArg, "--get-id", "--no-playlist")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("yt-dlp search failed: %w", err)
	}

	// Parse video IDs from output
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var videoURLs []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Convert video ID to full URL
		videoURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", line)

		// If channels are specified, filter by channel
		if len(s.channels) > 0 {
			if s.isVideoFromAllowedChannel(videoURL) {
				videoURLs = append(videoURLs, videoURL)
			}
		} else {
			videoURLs = append(videoURLs, videoURL)
		}

		// Stop if we have enough videos
		if len(videoURLs) >= s.maxVideos {
			break
		}
	}

	return videoURLs, nil
}

// isVideoFromAllowedChannel checks if a video is from one of the allowed channels
func (s *SearchQuerySource) isVideoFromAllowedChannel(videoURL string) bool {
	// Get video info to extract channel information
	cmd := exec.Command(s.ytDlpPath, videoURL, "--get", "channel_id", "--get", "channel")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("[SearchSource] Error getting channel info for %s: %v", videoURL, err)
		return false
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) < 2 {
		return false
	}

	channelID := strings.TrimSpace(lines[0])
	channelName := strings.TrimSpace(lines[1])

	// Check if channel ID or name matches any allowed channels
	for _, allowedChannel := range s.channels {
		allowedChannel = strings.TrimSpace(allowedChannel)

		// Check by channel ID
		if channelID == allowedChannel {
			return true
		}

		// Check by channel name (case-insensitive)
		if strings.EqualFold(channelName, allowedChannel) {
			return true
		}
	}

	return false
}
