package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type TestMigrationsCommand struct{}

func (c *TestMigrationsCommand) Name() string {
	return "test-migrations"
}

func (c *TestMigrationsCommand) Description() string {
	return "Test database migrations (up/down/idempotency)"
}

func (c *TestMigrationsCommand) Run(args []string) error {
	PrintHeader("Testing database migrations...")

	// Configuration
	dbName := "brandish_test_migrations"
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}
	dbPass := os.Getenv("DB_PASSWORD")
	if dbPass == "" {
		dbPass = "postgres"
	}
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}

	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPass, dbHost, dbPort, dbName)

	// Check if goose is installed
	if _, err := exec.LookPath("goose"); err != nil {
		PrintError("goose is not installed")
		fmt.Println("Install with: go install github.com/pressly/goose/v3/cmd/goose@latest")
		return fmt.Errorf("goose not installed")
	}

	// Setup cleanup
	defer func() {
		PrintInfo("Cleaning up test database...")
		_ = runCommand("psql", "-h", dbHost, "-p", dbPort, "-U", dbUser, "-c", fmt.Sprintf("DROP DATABASE IF EXISTS %s;", dbName))
	}()

	// Create test database
	PrintInfo("Creating test database: %s", dbName)
	_ = runCommand("psql", "-h", dbHost, "-p", dbPort, "-U", dbUser, "-c", fmt.Sprintf("DROP DATABASE IF EXISTS %s;", dbName))

	if err := runCommand("psql", "-h", dbHost, "-p", dbPort, "-U", dbUser, "-c", fmt.Sprintf("CREATE DATABASE %s;", dbName)); err != nil {
		PrintError("Error creating database: %v", err)
		return fmt.Errorf("database creation failed")
	}

	// Set environment variables for goose to avoid passing secrets in arguments
	os.Setenv("GOOSE_DRIVER", "postgres")
	os.Setenv("GOOSE_DBSTRING", dbURL)

	// Test UP migrations
	PrintInfo("Testing UP migrations...")
	if err := runCommandVerbose("goose", "-dir", "migrations", "up"); err != nil {
		PrintError("Error running UP migrations: %v", err)
		return fmt.Errorf("migrations up failed")
	}

	// Verify UP
	versionUp, err := getGooseVersion()
	if err != nil {
		PrintError("Error getting goose version: %v", err)
		return fmt.Errorf("failed to get goose version")
	}
	if strings.Contains(versionUp, "version 0") {
		PrintError("UP migrations did not update version (version is 0)")
		return fmt.Errorf("migrations up failed (version 0)")
	}
	PrintSuccess("UP migrations completed (Version: %s)", versionUp)

	// Test DOWN migrations (all the way)
	PrintInfo("Testing DOWN migrations (all)...")
	if err := runCommandVerbose("goose", "-dir", "migrations", "down-to", "0"); err != nil {
		PrintError("Error running DOWN migrations: %v", err)
		return fmt.Errorf("migrations down failed")
	}

	// Verify DOWN
	versionDown, err := getGooseVersion()
	if err != nil {
		PrintError("Error getting goose version: %v", err)
		return fmt.Errorf("failed to get goose version")
	}
	if !strings.Contains(versionDown, "version 0") {
		PrintError("DOWN migrations did not reset version (Version: %s)", versionDown)
		return fmt.Errorf("migrations down failed (version != 0)")
	}
	PrintSuccess("DOWN migrations completed (Version: %s)", versionDown)

	// Test UP migrations again (idempotency)
	PrintInfo("Testing UP migrations again (idempotency)...")
	if err := runCommandVerbose("goose", "-dir", "migrations", "up"); err != nil {
		PrintError("Error running UP migrations again: %v", err)
		return fmt.Errorf("migrations up (idempotency) failed")
	}

	// Verify Idempotency
	versionReUp, err := getGooseVersion()
	if err != nil {
		PrintError("Error getting goose version: %v", err)
		return fmt.Errorf("failed to get goose version")
	}
	if versionReUp != versionUp {
		PrintError("Migration count/version mismatch (%s vs %s)", versionUp, versionReUp)
		return fmt.Errorf("idempotency check failed")
	}
	PrintSuccess("UP migrations completed again (Version: %s)", versionReUp)

	// Final verification
	PrintSuccess("All migration tests passed!")
	return nil
}

func getGooseVersion() (string, error) {
	// getCommandOutput uses exec.Command which inherits env vars
	out, err := getCommandOutput("goose", "-dir", "migrations", "version")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}
