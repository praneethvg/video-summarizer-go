package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"video-summarizer-go/internal/api"
	"video-summarizer-go/internal/config"
	"video-summarizer-go/internal/core"
	"video-summarizer-go/internal/logging"
	"video-summarizer-go/internal/services"
	"video-summarizer-go/internal/sources"
)

func main() {
	if err := logging.SetupLogging("logging.yaml"); err != nil {
		panic(err)
	}

	serviceConfigPath := flag.String("service-config", "service.yaml", "Path to service configuration file")
	flag.Parse()

	// Load service configuration
	serviceCfg, err := config.LoadServiceConfig(*serviceConfigPath)
	if err != nil {
		log.Fatalf("Failed to load service config: %v", err)
	}

	// Load application configuration
	appCfg, err := config.LoadConfig(serviceCfg.EngineConfigPath)
	if err != nil {
		log.Fatalf("Failed to load app config: %v", err)
	}

	// Initialize core pipeline using SetupEngine
	engine, _, promptManager, err := core.SetupEngine(appCfg)
	if err != nil {
		log.Fatalf("Failed to set up engine: %v", err)
	}

	// Initialize video submission service
	submissionService := services.NewVideoSubmissionService(engine)

	// Initialize video source manager
	sourceManager := sources.NewVideoSourceManager()

	// Initialize API handler
	apiHandler := api.NewAPIHandler(submissionService, promptManager, sourceManager)

	// Set up HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("/api/submit", apiHandler.SubmitVideo)
	mux.HandleFunc("/api/status", apiHandler.GetStatus)
	mux.HandleFunc("/api/cancel", apiHandler.CancelRequest)
	mux.HandleFunc("/api/health", apiHandler.Health)
	mux.HandleFunc("/api/prompts", apiHandler.ListPrompts)

	// Create source factory
	sourceFactory := sources.NewSourceFactory(submissionService)

	// Add sources from configuration
	for _, sourceConfig := range serviceCfg.BackgroundSources.Sources {
		if !sourceConfig.Enabled {
			log.Warnf("Skipping disabled source: %s", sourceConfig.Name)
			continue
		}

		source, err := sourceFactory.CreateSource(&sourceConfig, appCfg.YtDlpPath)
		if err != nil {
			log.Errorf("Failed to create source %s: %v", sourceConfig.Name, err)
			continue
		}

		interval, err := sourceConfig.GetIntervalDuration()
		if err != nil {
			log.Errorf("Invalid interval for source %s: %v", sourceConfig.Name, err)
			continue
		}

		sourceManager.AddSource(sourceConfig.Name, source, sources.VideoSourceConfig{
			Enabled:   sourceConfig.Enabled,
			Interval:  interval,
			MaxVideos: sourceConfig.MaxVideosPerRun,
		})

		log.Infof("Added source: %s (type: %s, interval: %s)", sourceConfig.Name, sourceConfig.Type, sourceConfig.Interval)
	}

	// Create HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", serviceCfg.Server.Host, serviceCfg.Server.Port),
		Handler: mux,
	}

	// Start background video sources
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := sourceManager.StartAll(ctx); err != nil {
		log.Warnf("Failed to start some video sources: %v", err)
	}

	// Start the HTTP server in a goroutine
	go func() {
		log.Infof("Starting HTTP server on %s:%d", serviceCfg.Server.Host, serviceCfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Start the processing engine
	go func() {
		log.Println("Starting processing engine...")
		engine.Start()
	}()

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Stop video sources
	if err := sourceManager.StopAll(); err != nil {
		log.Errorf("Error stopping video sources: %v", err)
	}

	// Stop HTTP server
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Errorf("Error shutting down HTTP server: %v", err)
	}

	// Stop processing engine
	engine.Stop()

	log.Println("Shutdown complete")
}
