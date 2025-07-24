package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"video-summarizer-go/internal/config"
	"video-summarizer-go/internal/core"
	"video-summarizer-go/internal/interfaces"

	log "github.com/sirupsen/logrus"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to config file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Errorf("Failed to load config: %v", err)
		os.Exit(1)
	}

	engine, _, _, err := core.SetupEngine(cfg)
	if err != nil {
		log.Errorf("Failed to set up engine: %v", err)
		os.Exit(1)
	}

	videoURL := "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
	log.Infof("Submitting job for: %s", videoURL)

	requestID := fmt.Sprintf("demo-%d", time.Now().Unix())
	// Start processing
	prompt := interfaces.Prompt{Type: interfaces.PromptTypeID, Prompt: "market_report"}
	sourceType := "video"
	category := "general"
	maxTokens := 10000
	err = engine.StartRequest(requestID, videoURL, prompt, sourceType, category, maxTokens)
	if err != nil {
		log.Errorf("Failed to start request: %v", err)
		return
	}

	log.Infof("Processing started. Waiting for completion...")
	time.Sleep(30 * time.Second)

	state, err := engine.GetRequestState(requestID)
	if err != nil {
		log.Errorf("Failed to get final state: %v", err)
		return
	}

	log.Debugf("=== Final State ===")
	log.Infof("Request ID: %s", state.RequestID)
	log.Infof("Status: %s", state.Status)
	log.Infof("Error: %s", state.Error)

	if state.VideoInfo != nil {
		log.Debugf("=== Video Info ===")
		if title, ok := state.VideoInfo["title"].(string); ok {
			log.Infof("Title: %s", title)
		}
		if duration, ok := state.VideoInfo["duration"].(float64); ok {
			log.Infof("Duration: %.2f seconds", duration)
		}
		if uploader, ok := state.VideoInfo["uploader"].(string); ok {
			log.Infof("Uploader: %s", uploader)
		}
		if viewCount, ok := state.VideoInfo["view_count"].(float64); ok {
			log.Infof("View Count: %.0f", viewCount)
		}
	}

	if state.AudioPath != "" {
		log.Debugf("=== Audio ===")
		log.Infof("Audio Path: %s", state.AudioPath)
	}

	if state.Transcript != "" {
		log.Debugf("=== Transcript ===")
		log.Infof("Transcript: %s", state.Transcript)
	}

	if state.Summary != "" {
		log.Debugf("=== Summary ===")
		log.Infof("Summary: %s", state.Summary)
	}

	if state.OutputPath != "" {
		log.Debugf("=== Output ===")
		log.Infof("Output Path: %s", state.OutputPath)
	}

	log.Debugf("Demo complete.")
}
