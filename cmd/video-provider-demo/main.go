package main

import (
	"flag"
	"fmt"
	"os"

	"video-summarizer-go/internal/providers/video"

	log "github.com/sirupsen/logrus"
)

func main() {
	var url = flag.String("url", "", "YouTube URL to process")
	flag.Parse()

	if *url == "" {
		fmt.Println("Please provide a YouTube URL with --url")
		os.Exit(1)
	}

	provider := video.NewYtDlpVideoProvider("tools/yt-dlp", os.TempDir())

	log.Infof("Getting video info for: %s", *url)
	info, err := provider.GetVideoInfo(*url)
	if err != nil {
		log.Errorf("Error getting video info: %v", err)
		os.Exit(1)
	}

	log.Debugf("Video Info:")
	log.Infof("Title: %s", info["title"])
	log.Infof("Duration: %.2f seconds", info["duration"])
	log.Infof("Uploader: %s", info["uploader"])
	log.Infof("View Count: %.0f", info["view_count"])

	log.Debugf("Downloading audio...")
	audioPath, err := provider.DownloadAudio(*url)
	if err != nil {
		log.Errorf("Error downloading audio: %v", err)
		os.Exit(1)
	}

	log.Infof("Audio downloaded to: %s", audioPath)
}
