package logging

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	File   string `yaml:"file"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
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
		file, err := os.OpenFile(cfg.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			log.SetOutput(file)
		} else {
			log.Warn("Failed to log to file, using default stderr")
		}
	}
	return nil
}
