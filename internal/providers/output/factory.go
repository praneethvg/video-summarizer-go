package output

import (
	"fmt"

	"video-summarizer-go/internal/config"
	"video-summarizer-go/internal/interfaces"
)

func NewOutputProviderFromConfig(cfg *config.AppConfig) (interfaces.OutputProvider, error) {
	switch cfg.OutputProvider {
	case "gdrive":
		return NewGDriveOutputProvider(cfg)
	case "":
		return nil, fmt.Errorf("output_provider not set in config")
	default:
		return nil, fmt.Errorf("unsupported output provider: %s", cfg.OutputProvider)
	}
}
