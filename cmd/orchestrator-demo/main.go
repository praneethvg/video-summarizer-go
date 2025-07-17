package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"video-summarizer-go/internal/config"
	"video-summarizer-go/internal/core"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to config file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Println("Failed to load config:", err)
		os.Exit(1)
	}

	engine, _, _, err := core.SetupEngine(cfg)
	if err != nil {
		fmt.Println("Failed to set up engine:", err)
		os.Exit(1)
	}

	videoURL := "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
	fmt.Println("Submitting job for:", videoURL)

	requestID := fmt.Sprintf("demo-%d", time.Now().Unix())
	err = engine.StartRequest(requestID, videoURL)
	if err != nil {
		fmt.Println("Failed to start request:", err)
		return
	}

	fmt.Println("Processing started. Waiting for completion...")
	time.Sleep(30 * time.Second)

	state, err := engine.GetRequestState(requestID)
	if err != nil {
		fmt.Println("Failed to get final state:", err)
		return
	}

	fmt.Printf("\n=== Final State ===\n")
	fmt.Printf("Request ID: %s\n", state.RequestID)
	fmt.Printf("Status: %s\n", state.Status)
	fmt.Printf("Error: %s\n", state.Error)

	if state.VideoInfo != nil {
		fmt.Printf("\n=== Video Info ===\n")
		if title, ok := state.VideoInfo["title"].(string); ok {
			fmt.Printf("Title: %s\n", title)
		}
		if duration, ok := state.VideoInfo["duration"].(float64); ok {
			fmt.Printf("Duration: %.2f seconds\n", duration)
		}
		if uploader, ok := state.VideoInfo["uploader"].(string); ok {
			fmt.Printf("Uploader: %s\n", uploader)
		}
		if viewCount, ok := state.VideoInfo["view_count"].(float64); ok {
			fmt.Printf("View Count: %.0f\n", viewCount)
		}
	}

	if state.AudioPath != "" {
		fmt.Printf("\n=== Audio ===\n")
		fmt.Printf("Audio Path: %s\n", state.AudioPath)
	}

	if state.Transcript != "" {
		fmt.Printf("\n=== Transcript ===\n")
		fmt.Printf("Transcript: %s\n", state.Transcript)
	}

	if state.Summary != "" {
		fmt.Printf("\n=== Summary ===\n")
		fmt.Printf("Summary: %s\n", state.Summary)
	}

	if state.OutputPath != "" {
		fmt.Printf("\n=== Output ===\n")
		fmt.Printf("Output Path: %s\n", state.OutputPath)
	}

	fmt.Printf("\nDemo complete.\n")
}
