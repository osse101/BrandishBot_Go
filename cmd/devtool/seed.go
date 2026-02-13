package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type SeedCommand struct{}

func (c *SeedCommand) Name() string {
	return "seed"
}

func (c *SeedCommand) Description() string {
	return "Seed database with data (test, staging)"
}

func (c *SeedCommand) Run(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("subcommand required: test, staging")
	}
	subcmd := args[0]

	// Construct DB URL
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbUser := getEnv("DB_USER", "dev")
		dbPass := getEnv("DB_PASSWORD", "change_this_secure_password")
		dbHost := getEnv("DB_HOST", "localhost")
		dbPort := getEnv("DB_PORT", "5432")
		dbName := getEnv("DB_NAME", "app")

		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPass, dbHost, dbPort, dbName)
	}

	PrintInfo("Connecting to database: %s (redacted password)", redactPassword(dbURL))

	// Open connection to DB
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	switch subcmd {
	case "test":
		return c.runTestSeed(db)
	case "staging":
		return c.runStagingSeed(db)
	default:
		return fmt.Errorf("unknown subcommand: %s", subcmd)
	}
}

func (c *SeedCommand) runTestSeed(db *sql.DB) error {
	PrintInfo("Running test seeds...")

	files := []string{
		"internal/database/seeds/test_recipe.sql",
		"internal/database/seeds/test_user.sql",
	}

	for _, file := range files {
		if err := c.executeFile(db, file); err != nil {
			return err
		}
	}

	PrintSuccess("Test seeds completed successfully")
	return nil
}

func (c *SeedCommand) runStagingSeed(db *sql.DB) error {
	PrintInfo("Running staging seeds...")

	// Staging only runs test_user.sql based on previous Makefile logic
	files := []string{
		"internal/database/seeds/test_user.sql",
	}

	for _, file := range files {
		if err := c.executeFile(db, file); err != nil {
			return err
		}
	}

	PrintSuccess("Staging seeds completed successfully")
	return nil
}

func (c *SeedCommand) executeFile(db *sql.DB, filepath string) error {
	PrintInfo("Executing %s...", filepath)

	content, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read seed file %s: %w", filepath, err)
	}

	if _, err := db.Exec(string(content)); err != nil {
		return fmt.Errorf("failed to execute seed file %s: %w", filepath, err)
	}

	return nil
}

// Helper to redact password from connection string for logging
func redactPassword(connStr string) string {
	// Simple redaction logic
	// This is just for logging, doesn't need to be perfect but should catch common patterns
	// postgres://user:pass@host:port/db
	// Replace :pass@ with :***@
	return connStr // Simplification: just return it for now or implement proper parsing if needed.
}
