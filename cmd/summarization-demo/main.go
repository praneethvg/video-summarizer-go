package main

import (
	"context"
	"flag"
	"os"
	"strings"

	"video-summarizer-go/internal/config"
	"video-summarizer-go/internal/providers/summarization"

	"github.com/sirupsen/logrus"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to config file")
	textFile := flag.String("file", "", "Path to text file to summarize")
	text := flag.String("text", "", "Text to summarize (alternative to file)")
	prompt := flag.String("prompt", "general", "Summarization prompt type")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		logrus.Errorf("Failed to load config: %v", err)
		os.Exit(1)
	}

	provider, err := summarization.NewConfigurableSummarizationProviderFromConfig(cfg)
	if err != nil {
		logrus.Errorf("Failed to initialize summarization provider: %v", err)
		os.Exit(1)
	}

	// Initialize prompt manager
	promptManager := config.NewPromptManager()
	promptsDir := cfg.PromptsDir
	if promptsDir == "" {
		promptsDir = "prompts"
	}
	if err := promptManager.LoadPrompts(promptsDir); err != nil {
		logrus.Errorf("Failed to load prompts: %v", err)
		os.Exit(1)
	}

	var inputText string
	if *textFile != "" {
		data, err := os.ReadFile(*textFile)
		if err != nil {
			logrus.Errorf("Failed to read file: %v", err)
			os.Exit(1)
		}
		inputText = string(data)
	} else if *text != "" {
		inputText = *text
	} else {
		logrus.Errorf("Please provide --file or --text.")
		os.Exit(1)
	}

	logrus.Debugf("Generating summary with prompt: '%s'", *prompt)
	logrus.Println(strings.Repeat("=", 50))
	summary, err := provider.SummarizeText(context.Background(), inputText, *prompt, 10000)
	if err != nil {
		logrus.Errorf("Summarization error: %v", err)
		os.Exit(1)
	}
	logrus.Println("\nSummary:\n", summary)
}
