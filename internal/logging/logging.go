package logging

import (
	"io/ioutil"
	"os"

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
		log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	}

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
