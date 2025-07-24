package config

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type AppConfig struct {
	SummarizerProvider string `yaml:"summarizer_provider"`
	OpenAIKey          string `yaml:"openai_api_key"`
	OpenAIModel        string `yaml:"openai_model"`
	OpenAIMaxTokens    int    `yaml:"openai_max_tokens"`
	YtDlpPath          string `yaml:"yt_dlp_path"`
	WhisperPath        string `yaml:"whisper_path"`
	WhisperModelPath   string `yaml:"whisper_model_path"`
	TmpDir             string `yaml:"tmp_dir"`
	PromptsDir         string `yaml:"prompts_dir"`

	OutputProvider    string `yaml:"output_provider"`
	GDriveAuthMethod  string `yaml:"gdrive_auth_method"`
	GDriveCredentials string `yaml:"gdrive_credentials_file"`
	GDriveToken       string `yaml:"gdrive_token_file"`
	GDriveFolderID    string `yaml:"gdrive_folder_id"`
	UploadSummary     bool   `yaml:"upload_summary"`
	UploadTranscript  bool   `yaml:"upload_transcript"`

	// Concurrency limits for worker pools
	Concurrency ConcurrencyConfig `yaml:"concurrency"`
}

// ConcurrencyConfig defines concurrency limits for different task types
type ConcurrencyConfig struct {
	Transcription int `yaml:"transcription"`
	Summarization int `yaml:"summarization"`
	VideoInfo     int `yaml:"video_info"`
	Output        int `yaml:"output"`
	Cleanup       int `yaml:"cleanup"`
	AudioDownload int `yaml:"audio_download"`
}

func LoadConfig(path string) (*AppConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer f.Close()
	var cfg AppConfig
	dec := yaml.NewDecoder(f)
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	// Debug logging
	log.Infof("Loaded config from %s", path)
	log.Infof("OpenAI model: %s", cfg.OpenAIModel)
	log.Infof("Summarizer provider: %s", cfg.SummarizerProvider)

	return &cfg, nil
}
