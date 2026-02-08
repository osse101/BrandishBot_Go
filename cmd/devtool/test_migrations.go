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

	config := getMigrationTestConfig()

	if err := verifyGooseInstalled(); err != nil {
		return err
	}

	defer cleanupTestDatabase(config)

	if err := setupTestDatabase(config); err != nil {
		return err
	}

	os.Setenv("GOOSE_DRIVER", "postgres")
	os.Setenv("GOOSE_DBSTRING", config.dbURL)

	versionUp, err := runMigrationUpTests()
	if err != nil {
		return err
	}

	if err := runMigrationDownTests(); err != nil {
		return err
	}

	if err := runMigrationIdempotencyTests(versionUp); err != nil {
		return err
	}

	PrintSuccess("All migration tests passed!")
	return nil
}

type migrationTestConfig struct {
	dbName string
	dbHost string
	dbPort string
	dbUser string
	dbURL  string
}

func getMigrationTestConfig() migrationTestConfig {
	dbName := "brandish_test_migrations"
	dbUser := getEnv("DB_USER", "postgres")
	dbPass := getEnv("DB_PASSWORD", "postgres")
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")

	return migrationTestConfig{
		dbName: dbName,
		dbHost: dbHost,
		dbPort: dbPort,
		dbUser: dbUser,
		dbURL:  fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPass, dbHost, dbPort, dbName),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func verifyGooseInstalled() error {
	if _, err := exec.LookPath("goose"); err != nil {
		PrintError("goose is not installed")
		fmt.Println("Install with: go install github.com/pressly/goose/v3/cmd/goose@latest")
		return fmt.Errorf("goose not installed")
	}
	return nil
}

func cleanupTestDatabase(config migrationTestConfig) {
	PrintInfo("Cleaning up test database...")
	_ = runCommand("psql", "-h", config.dbHost, "-p", config.dbPort, "-U", config.dbUser, "-c", fmt.Sprintf("DROP DATABASE IF EXISTS %s;", config.dbName))
}

func setupTestDatabase(config migrationTestConfig) error {
	PrintInfo("Creating test database: %s", config.dbName)
	_ = runCommand("psql", "-h", config.dbHost, "-p", config.dbPort, "-U", config.dbUser, "-c", fmt.Sprintf("DROP DATABASE IF EXISTS %s;", config.dbName))

	if err := runCommand("psql", "-h", config.dbHost, "-p", config.dbPort, "-U", config.dbUser, "-c", fmt.Sprintf("CREATE DATABASE %s;", config.dbName)); err != nil {
		PrintError("Error creating database: %v", err)
		return fmt.Errorf("database creation failed")
	}
	return nil
}

func runMigrationUpTests() (string, error) {
	PrintInfo("Testing UP migrations...")
	if err := runCommandVerbose("goose", "-dir", "migrations", "up"); err != nil {
		PrintError("Error running UP migrations: %v", err)
		return "", fmt.Errorf("migrations up failed")
	}

	versionUp, err := getGooseVersion()
	if err != nil {
		PrintError("Error getting goose version: %v", err)
		return "", fmt.Errorf("failed to get goose version")
	}
	if strings.Contains(versionUp, "version 0") {
		PrintError("UP migrations did not update version (version is 0)")
		return "", fmt.Errorf("migrations up failed (version 0)")
	}
	PrintSuccess("UP migrations completed (Version: %s)", versionUp)
	return versionUp, nil
}

func runMigrationDownTests() error {
	PrintInfo("Testing DOWN migrations (all)...")
	if err := runCommandVerbose("goose", "-dir", "migrations", "down-to", "0"); err != nil {
		PrintError("Error running DOWN migrations: %v", err)
		return fmt.Errorf("migrations down failed")
	}

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
	return nil
}

func runMigrationIdempotencyTests(expectedVersion string) error {
	PrintInfo("Testing UP migrations again (idempotency)...")
	if err := runCommandVerbose("goose", "-dir", "migrations", "up"); err != nil {
		PrintError("Error running UP migrations again: %v", err)
		return fmt.Errorf("migrations up (idempotency) failed")
	}

	versionReUp, err := getGooseVersion()
	if err != nil {
		PrintError("Error getting goose version: %v", err)
		return fmt.Errorf("failed to get goose version")
	}
	if versionReUp != expectedVersion {
		PrintError("Migration count/version mismatch (%s vs %s)", expectedVersion, versionReUp)
		return fmt.Errorf("idempotency check failed")
	}
	PrintSuccess("UP migrations completed again (Version: %s)", versionReUp)
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
