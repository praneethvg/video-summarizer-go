package logging

import (
	"fmt"
	"io"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	File   string `yaml:"file"`
	// Log rotation settings
	MaxSize    int  `yaml:"max_size"`    // Maximum size in megabytes before rotation
	MaxBackups int  `yaml:"max_backups"` // Maximum number of old log files to retain
	MaxAge     int  `yaml:"max_age"`     // Maximum number of days to retain old log files
	Compress   bool `yaml:"compress"`    // Whether to compress rotated log files
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	// Set defaults for log rotation if not specified
	if cfg.MaxSize == 0 {
		cfg.MaxSize = 100 // 100 MB default
	}
	if cfg.MaxBackups == 0 {
		cfg.MaxBackups = 3 // Keep 3 backup files default
	}
	if cfg.MaxAge == 0 {
		cfg.MaxAge = 28 // Keep logs for 28 days default
	}
	return &cfg, nil
}

// Custom formatter to put log message first, then file:line and method
// e.g. INFO[time] My log message | core/engine.go:302 (*ProcessingEngine).WorkerProcess

type MessageFirstFormatter struct {
	log.TextFormatter
}

func (f *MessageFirstFormatter) Format(entry *log.Entry) ([]byte, error) {
	// Use the standard formatter to get the prefix (level, time, etc.)
	prefix := fmt.Sprintf("%s[%s] ", strings.ToUpper(entry.Level.String()), entry.Time.Format(f.TimestampFormat))
	msg := entry.Message
	var caller string
	if entry.HasCaller() {
		relFile := entry.Caller.File
		if idx := strings.Index(relFile, "video-summarizer-go/"); idx != -1 {
			relFile = relFile[idx+len("video-summarizer-go/"):]
		}
		funcName := entry.Caller.Function
		if slash := strings.LastIndex(funcName, "/"); slash != -1 {
			funcName = funcName[slash+1:]
		}
		caller = fmt.Sprintf(" | %s:%d %s", relFile, entry.Caller.Line, funcName)
	}
	return []byte(fmt.Sprintf("%s%s%s\n", prefix, msg, caller)), nil
}

func SetupLogging(path string) error {
	cfg, err := LoadConfig(path)
	if err != nil {
		return err
	}
	// Set log level
	level, err := log.ParseLevel(cfg.Level)
	if err != nil {
		level = log.InfoLevel
	}
	log.SetLevel(level)

	// Set log format
	switch cfg.Format {
	case "json":
		log.SetFormatter(&log.JSONFormatter{})
	default:
		log.SetFormatter(&MessageFirstFormatter{
			TextFormatter: log.TextFormatter{
				FullTimestamp:   true,
				TimestampFormat: "2006-01-02T15:04:05-07:00",
				DisableQuote:    true,
			},
		})
	}

	// Enable caller reporting
	log.SetReportCaller(true)

	// Set log output
	if cfg.File != "" {
		lumberjackLogger := &lumberjack.Logger{
			Filename:   cfg.File,
			MaxSize:    cfg.MaxSize, // MB
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge, // days
			Compress:   cfg.Compress,
		}
		log.SetOutput(io.MultiWriter(os.Stderr, lumberjackLogger))
	} else {
		log.SetOutput(io.MultiWriter(os.Stderr))
	}
	return nil
}
