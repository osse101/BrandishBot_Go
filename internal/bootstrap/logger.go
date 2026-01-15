package bootstrap

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/config"
)

// SetupLogger initializes the application logger with file and stdout output.
// It creates the log directory, cleans up old logs, sets up a MultiWriter for
// stdout and file output, parses the log level, and initializes slog.
// Returns the log file handle (caller must close) and any error encountered.
func SetupLogger(cfg *config.Config) (*os.File, error) {
	// Create logs directory
	if err := os.MkdirAll(cfg.LogDir, DirPermission); err != nil {
		return nil, fmt.Errorf("%s: %w", LogMsgFailedCreateLogsDir, err)
	}

	// Cleanup old logs (keep 9 most recent)
	cleanupLogs(cfg.LogDir)

	// Create timestamped log file
	timestamp := time.Now().Format(LogFileTimestampFormat)
	logFileName := filepath.Join(cfg.LogDir, fmt.Sprintf(LogFileNamePattern, timestamp))

	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, LogFilePermission)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", LogMsgFailedOpenLogFile, err)
	}

	// Initialize logger with MultiWriter (stdout + file)
	mw := io.MultiWriter(os.Stdout, logFile)

	// Parse log level from config
	var level slog.Level
	switch strings.ToUpper(cfg.LogLevel) {
	case LogLevelDebug:
		level = slog.LevelDebug
	case LogLevelWarn:
		level = slog.LevelWarn
	case LogLevelError:
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Create and set default logger
	logger := slog.New(slog.NewTextHandler(mw, &slog.HandlerOptions{
		Level: level,
	}))
	slog.SetDefault(logger)

	// Log initialization messages
	slog.Info(LogMsgLoggingInitialized, "level", level)
	slog.Info(LogMsgStartingBrandishBot,
		"environment", cfg.Environment,
		"log_level", cfg.LogLevel,
		"log_format", cfg.LogFormat,
		"version", cfg.Version)

	slog.Debug(LogMsgConfigurationLoaded,
		"db_host", cfg.DBHost,
		"db_port", cfg.DBPort,
		"db_name", cfg.DBName,
		"port", cfg.Port)

	return logFile, nil
}

// cleanupLogs removes old log files, keeping only the 9 most recent.
// This prevents unbounded log file accumulation.
func cleanupLogs(logDir string) {
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return
	}

	var logFiles []os.DirEntry
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), LogFileExtension) {
			logFiles = append(logFiles, entry)
		}
	}

	if len(logFiles) >= LogFileRetentionLimit {
		// Delete oldest files until we have 9 left
		toDelete := len(logFiles) - LogFileRetentionCount
		for i := 0; i < toDelete; i++ {
			err := os.Remove(filepath.Join(logDir, logFiles[i].Name()))
			if err != nil {
				fmt.Printf(LogMsgFailedDeleteOldLog, logFiles[i].Name(), err)
			}
		}
	}
}
