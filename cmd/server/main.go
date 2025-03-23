package main

import (
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"

	"shodone/internal/api"
	"shodone/internal/config"
	"shodone/internal/storage"
)

func main() {
	// Initialize logger
	logger := log.New()
	logger.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
	logger.SetOutput(os.Stdout)
	// Set log level, first try to parse from environment
	if level, err := log.ParseLevel(os.Getenv("SHODONE_LOG_LEVEL")); err == nil {
		logger.SetLevel(level)
	} else {
		logger.SetLevel(log.InfoLevel)
		logger.Warn("Invalid log level, defaulting to info")
	}

	// Load configuration
	cfg, err := config.New()
	if err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	db, err := storage.New(cfg.DatabasePath)
	if err != nil {
		logger.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize and start API server
	server := api.NewServer(cfg, db, logger)
	go func() {
		if err := server.Start(); err != nil {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Handle graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")
	if err := server.Stop(); err != nil {
		logger.Fatalf("Server shutdown failed: %v", err)
	}
	logger.Info("Server stopped")
}
