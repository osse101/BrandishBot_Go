package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type CheckDBCommand struct{}

func (c *CheckDBCommand) Name() string {
	return "check-db"
}

func (c *CheckDBCommand) Description() string {
	return "Check if database is running and ready"
}

func (c *CheckDBCommand) Run(args []string) error {
	PrintHeader("Checking Docker database status...")

	// Check if docker compose is available
	if err := runCommand("docker", "compose", "version"); err != nil {
		return fmt.Errorf("docker compose not found. Please install Docker Compose")
	}

	// Check if db service is running
	out, err := getCommandOutput("docker", "compose", "ps", "db")
	running := false
	if err == nil {
		status := strings.ToLower(out)
		if strings.Contains(status, "up") || strings.Contains(status, "running") {
			running = true
		}
	}

	if running {
		PrintSuccess("Database is already running")
	} else {
		PrintInfo("Starting database...")
		if err := runCommandVerbose("docker", "compose", "up", "-d", "db"); err != nil {
			return fmt.Errorf("error starting database: %v", err)
		}

		PrintInfo("Waiting for database to be ready...")
		time.Sleep(3 * time.Second)

		// Wait for database to accept connections
		maxAttempts := 30
		dbUser := os.Getenv("DB_USER")
		if dbUser == "" {
			dbUser = "dev"
		}
		dbName := os.Getenv("DB_NAME")
		if dbName == "" {
			dbName = "app"
		}

		ready := false
		for attempt := 0; attempt < maxAttempts; attempt++ {
			err := runCommand("docker", "compose", "exec", "-T", "db", "pg_isready", "-U", dbUser, "-d", dbName)
			if err == nil {
				PrintSuccess("Database is ready")
				ready = true
				break
			}

			if attempt == maxAttempts-1 {
				PrintError("Database failed to start after 30 seconds")
				_ = runCommandVerbose("docker", "compose", "logs", "db")
				return fmt.Errorf("database failed to start")
			}

			fmt.Printf("Waiting for database... (%d/%d)\n", attempt+1, maxAttempts)
			time.Sleep(1 * time.Second)
		}
		if !ready {
			return fmt.Errorf("database not ready")
		}
	}

	PrintSuccess("Database check complete")
	return nil
}
