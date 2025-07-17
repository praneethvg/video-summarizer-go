package main

import (
	"flag"
	"fmt"
	"os"

	"video-summarizer-go/internal/providers/video"
)

func main() {
	var url = flag.String("url", "", "YouTube URL to process")
	flag.Parse()

	if *url == "" {
		fmt.Println("Please provide a YouTube URL with --url")
		os.Exit(1)
	}

	provider := video.NewYtDlpVideoProvider("tools/yt-dlp", os.TempDir())

	fmt.Printf("Getting video info for: %s\n", *url)
	info, err := provider.GetVideoInfo(*url)
	if err != nil {
		fmt.Printf("Error getting video info: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nVideo Info:\n")
	fmt.Printf("Title: %s\n", info["title"])
	fmt.Printf("Duration: %.2f seconds\n", info["duration"])
	fmt.Printf("Uploader: %s\n", info["uploader"])
	fmt.Printf("View Count: %.0f\n", info["view_count"])

	fmt.Printf("\nDownloading audio...\n")
	audioPath, err := provider.DownloadAudio(*url)
	if err != nil {
		fmt.Printf("Error downloading audio: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Audio downloaded to: %s\n", audioPath)
}
