package main

import (
	"flag"
	"fmt"
	"os"

	"video-summarizer-go/internal/providers/transcription"
)

func main() {
	var (
		audioPath   = flag.String("audio", "", "Path to audio file to transcribe")
		modelPath   = flag.String("model", "models/ggml-tiny.en.bin", "Path to whisper.cpp model file")
		whisperPath = flag.String("whisper", "tools/whisper", "Path to whisper.cpp binary")
	)
	flag.Parse()

	if *audioPath == "" {
		fmt.Println("Please provide --audio path to an audio file.")
		os.Exit(1)
	}

	provider := transcription.NewWhisperCppTranscriptionProvider(*whisperPath, *modelPath)
	fmt.Println("Transcribing:", *audioPath)
	transcript, err := provider.TranscribeAudio(*audioPath)
	if err != nil {
		fmt.Println("Transcription error:", err)
		os.Exit(1)
	}
	fmt.Println("\nTranscript:\n", transcript)
}
