package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"video-summarizer-go/internal/config"
	"video-summarizer-go/internal/providers/summarization"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to config file")
	textFile := flag.String("file", "", "Path to text file to summarize")
	text := flag.String("text", "", "Text to summarize (alternative to file)")
	prompt := flag.String("prompt", "general", "Summarization prompt type")
	listPrompts := flag.Bool("list-prompts", false, "List available prompts")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Println("Failed to load config:", err)
		os.Exit(1)
	}

	provider, err := summarization.NewConfigurableSummarizationProviderFromConfig(cfg)
	if err != nil {
		fmt.Println("Failed to initialize summarization provider:", err)
		os.Exit(1)
	}

	// Initialize prompt manager
	promptManager := config.NewPromptManager()
	promptsDir := cfg.PromptsDir
	if promptsDir == "" {
		promptsDir = "prompts"
	}
	if err := promptManager.LoadPrompts(promptsDir); err != nil {
		fmt.Printf("Failed to load prompts: %v\n", err)
		os.Exit(1)
	}

	// Set prompt manager on the provider
	if openaiProvider, ok := provider.(*summarization.OpenAISummarizationProvider); ok {
		openaiProvider.SetPromptManager(promptManager)
	} else if textProvider, ok := provider.(*summarization.TextSummarizationProvider); ok {
		textProvider.SetPromptManager(promptManager)
	}

	if *listPrompts {
		fmt.Println("Available prompts:")
		for _, prompt := range provider.GetAvailablePrompts() {
			fmt.Printf("  - %s\n", prompt)
		}
		return
	}

	var inputText string
	if *textFile != "" {
		data, err := ioutil.ReadFile(*textFile)
		if err != nil {
			fmt.Println("Failed to read file:", err)
			os.Exit(1)
		}
		inputText = string(data)
	} else if *text != "" {
		inputText = *text
	} else {
		fmt.Println("Please provide --file or --text.")
		os.Exit(1)
	}

	fmt.Printf("\nGenerating summary with prompt: '%s'\n", *prompt)
	fmt.Println(strings.Repeat("=", 50))
	summary, err := provider.SummarizeText(context.Background(), inputText, *prompt, 10000)
	if err != nil {
		fmt.Println("Summarization error:", err)
		os.Exit(1)
	}
	fmt.Println("\nSummary:\n", summary)
}
