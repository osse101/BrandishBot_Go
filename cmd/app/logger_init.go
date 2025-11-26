package main

import (
	"github.com/osse101/BrandishBot_Go/internal/config"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// initLogger initializes the logger using centralized app configuration
func initLogger(cfg *config.Config) {
	// Determine if we should add source info (only in dev)
	addSource := cfg.Environment == "dev" || cfg.Environment == "development"
	
	loggerConfig := logger.NewConfig(
		cfg.LogLevel,
		cfg.LogFormat,
		cfg.ServiceName,
		cfg.Version,
		cfg.Environment,
		addSource,
	)
	
	logger.InitLogger(loggerConfig)
}
