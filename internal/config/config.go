package config

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

// AppConfig represents the main application configuration
type AppConfig struct {
	// Summarizer Provider
	SummarizerProvider string `yaml:"summarizer_provider"`

	// OpenAI Settings
	OpenAIKey       string `yaml:"openai_api_key"`
	OpenAIModel     string `yaml:"openai_model"`
	OpenAIMaxTokens int    `yaml:"openai_max_tokens"`

	// Video Provider
	YtDlpPath string `yaml:"yt_dlp_path"`

	// Transcription Provider
	WhisperPath      string `yaml:"whisper_path"`
	WhisperModelPath string `yaml:"whisper_model_path"`

	// Directories
	TmpDir     string `yaml:"tmp_dir"`
	PromptsDir string `yaml:"prompts_dir"`

	// Output Provider
	OutputProvider string `yaml:"output_provider"`

	// Google Drive Settings
	GDriveAuthMethod      string `yaml:"gdrive_auth_method"`
	GDriveCredentialsFile string `yaml:"gdrive_credentials_file"`
	GDriveTokenFile       string `yaml:"gdrive_token_file"`
	GDriveFolderID        string `yaml:"gdrive_folder_id"`
	UploadSummary         bool   `yaml:"upload_summary"`
	UploadTranscript      bool   `yaml:"upload_transcript"`

	// Concurrency
	Concurrency map[string]int `yaml:"concurrency"`
}

func LoadConfig(path string) (*AppConfig, error) {
	// Read YAML file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var cfg AppConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
	}

	// Apply environment variable overrides
	cfg.applyEnvOverrides()

	// Set defaults for missing values
	cfg.setDefaults()

	return &cfg, nil
}

// applyEnvOverrides applies environment variable overrides to the config
func (c *AppConfig) applyEnvOverrides() {
	// Helper function to get env var with fallback
	getEnv := func(key, fallback string) string {
		if val := os.Getenv(key); val != "" {
			return val
		}
		return fallback
	}

	getEnvBool := func(key string, fallback bool) bool {
		if val := os.Getenv(key); val != "" {
			if b, err := strconv.ParseBool(val); err == nil {
				return b
			}
		}
		return fallback
	}

	getEnvInt := func(key string, fallback int) int {
		if val := os.Getenv(key); val != "" {
			if i, err := strconv.Atoi(val); err == nil {
				return i
			}
		}
		return fallback
	}

	// Apply overrides
	c.SummarizerProvider = getEnv("VS_SUMMARIZER_PROVIDER", c.SummarizerProvider)
	c.OpenAIKey = getEnv("VS_OPENAI_API_KEY", c.OpenAIKey)
	c.OpenAIModel = getEnv("VS_OPENAI_MODEL", c.OpenAIModel)
	c.OpenAIMaxTokens = getEnvInt("VS_OPENAI_MAX_TOKENS", c.OpenAIMaxTokens)
	c.YtDlpPath = getEnv("VS_YT_DLP_PATH", c.YtDlpPath)
	c.WhisperPath = getEnv("VS_WHISPER_PATH", c.WhisperPath)
	c.WhisperModelPath = getEnv("VS_WHISPER_MODEL_PATH", c.WhisperModelPath)
	c.TmpDir = getEnv("VS_TMP_DIR", c.TmpDir)
	c.PromptsDir = getEnv("VS_PROMPTS_DIR", c.PromptsDir)
	c.OutputProvider = getEnv("VS_OUTPUT_PROVIDER", c.OutputProvider)
	c.GDriveAuthMethod = getEnv("VS_GDRIVE_AUTH_METHOD", c.GDriveAuthMethod)
	c.GDriveCredentialsFile = getEnv("VS_GDRIVE_CREDENTIALS_FILE", c.GDriveCredentialsFile)
	c.GDriveTokenFile = getEnv("VS_GDRIVE_TOKEN_FILE", c.GDriveTokenFile)
	c.GDriveFolderID = getEnv("VS_GDRIVE_FOLDER_ID", c.GDriveFolderID)
	c.UploadSummary = getEnvBool("VS_UPLOAD_SUMMARY", c.UploadSummary)
	c.UploadTranscript = getEnvBool("VS_UPLOAD_TRANSCRIPT", c.UploadTranscript)

	// Handle concurrency overrides
	c.applyConcurrencyOverrides()
}

// applyConcurrencyOverrides applies environment variable overrides for concurrency settings
func (c *AppConfig) applyConcurrencyOverrides() {
	// Initialize concurrency map if nil
	if c.Concurrency == nil {
		c.Concurrency = make(map[string]int)
	}

	// Define concurrency types and their environment variable names
	concurrencyTypes := map[string]string{
		"transcription":  "VS_CONCURRENCY_TRANSCRIPTION",
		"summarization":  "VS_CONCURRENCY_SUMMARIZATION",
		"video_info":     "VS_CONCURRENCY_VIDEO_INFO",
		"output":         "VS_CONCURRENCY_OUTPUT",
		"cleanup":        "VS_CONCURRENCY_CLEANUP",
		"audio_download": "VS_CONCURRENCY_AUDIO_DOWNLOAD",
	}

	// Apply overrides for each concurrency type
	for concurrencyType, envVar := range concurrencyTypes {
		if val := os.Getenv(envVar); val != "" {
			if i, err := strconv.Atoi(val); err == nil {
				c.Concurrency[concurrencyType] = i
			}
		}
	}
}

// setDefaults sets default values for missing configuration
func (c *AppConfig) setDefaults() {
	if c.SummarizerProvider == "" {
		c.SummarizerProvider = "openai"
	}
	if c.OpenAIModel == "" {
		c.OpenAIModel = "gpt-4o"
	}
	if c.OpenAIMaxTokens == 0 {
		c.OpenAIMaxTokens = 10000
	}
	if c.YtDlpPath == "" {
		c.YtDlpPath = "/app/tools/yt-dlp"
	}
	if c.WhisperPath == "" {
		c.WhisperPath = "/app/tools/whisper"
	}
	if c.WhisperModelPath == "" {
		c.WhisperModelPath = "/app/models/ggml-tiny.en.bin"
	}
	if c.TmpDir == "" {
		c.TmpDir = "/tmp"
	}
	if c.PromptsDir == "" {
		c.PromptsDir = "/app/prompts"
	}
	if c.OutputProvider == "" {
		c.OutputProvider = "gdrive"
	}
	if c.GDriveAuthMethod == "" {
		c.GDriveAuthMethod = "oauth"
	}
	if c.GDriveCredentialsFile == "" {
		c.GDriveCredentialsFile = "/app/secrets/gdrive_credentials.json"
	}
	if c.GDriveTokenFile == "" {
		c.GDriveTokenFile = "/app/secrets/gdrive_token.json"
	}
	if c.Concurrency == nil {
		c.Concurrency = map[string]int{
			"transcription":  2,
			"summarization":  3,
			"video_info":     1,
			"output":         1,
			"cleanup":        1,
			"audio_download": 1,
		}
	}
}
