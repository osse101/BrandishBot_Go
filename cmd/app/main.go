package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/osse101/BrandishBot_Go/internal/database"
	"github.com/osse101/BrandishBot_Go/internal/database/postgres"
	"github.com/osse101/BrandishBot_Go/internal/server"
	"github.com/osse101/BrandishBot_Go/internal/user"
)

func main() {
	// Load .env file first to get LOG_LEVEL
	if err := godotenv.Load(); err != nil {
		fmt.Println("No .env file found, using default/environment variables")
	}

	// Setup logging
	if err := os.MkdirAll("logs", 0755); err != nil {
		panic(fmt.Sprintf("Failed to create logs directory: %v", err))
	}

	// Cleanup old logs
	entries, err := os.ReadDir("logs")
	if err == nil {
		var logFiles []os.DirEntry
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".log") {
				logFiles = append(logFiles, entry)
			}
		}

		// Sort by name (which includes timestamp) descending to keep newest
		// Actually, standard string sort of YYYY-MM-DD_HH-MM-SS works for chronological order.
		// We want to keep the *last* 10.
		// Let's sort them.
		// Note: ReadDir returns sorted by filename, so they are already sorted by timestamp if format is correct.
		// We want to delete the oldest ones if count > 9 (leaving room for the new one to make 10).
		// Or just keep 10 total. Let's keep 9 existing + 1 new = 10.
		if len(logFiles) >= 10 {
			// Delete oldest files until we have 9 left
			toDelete := len(logFiles) - 9
			for i := 0; i < toDelete; i++ {
				err := os.Remove(filepath.Join("logs", logFiles[i].Name()))
				if err != nil {
					fmt.Printf("Failed to delete old log file %s: %v\n", logFiles[i].Name(), err)
				}
			}
		}
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logFileName := filepath.Join("logs", fmt.Sprintf("session_%s.log", timestamp))

	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(fmt.Sprintf("Failed to open log file: %v", err))
	}
	defer logFile.Close()

	mw := io.MultiWriter(os.Stdout, logFile)

	var level slog.Level
	switch strings.ToUpper(os.Getenv("LOG_LEVEL")) {
	case "DEBUG":
		level = slog.LevelDebug
	case "WARN":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	logger := slog.New(slog.NewTextHandler(mw, &slog.HandlerOptions{
		Level: level,
	}))
	slog.SetDefault(logger)

	slog.Info("Logging initialized", "level", level)

	// Construct connection string
	connString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	dbPool, err := database.NewPool(connString)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	userRepo := postgres.NewUserRepository(dbPool)
	userService := user.NewService(userRepo)
	srv := server.NewServer(8080, userService)

	if err := srv.Start(); err != nil {
		slog.Error("Server failed to start", "error", err)
		os.Exit(1)
	}
}
