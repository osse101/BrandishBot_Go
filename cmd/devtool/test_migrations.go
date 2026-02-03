package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func runTestMigrations() {
	fmt.Printf("%sTesting database migrations...%s\n", colorYellow, colorReset)

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
		fmt.Printf("%sError: goose is not installed%s\n", colorRed, colorReset)
		fmt.Println("Install with: go install github.com/pressly/goose/v3/cmd/goose@latest")
		os.Exit(1)
	}

	// Setup cleanup
	defer func() {
		fmt.Printf("%sCleaning up test database...%s\n", colorYellow, colorReset)
		_ = runCommand("psql", "-h", dbHost, "-p", dbPort, "-U", dbUser, "-c", fmt.Sprintf("DROP DATABASE IF EXISTS %s;", dbName))
	}()

	// Create test database
	fmt.Printf("%sCreating test database: %s%s\n", colorYellow, dbName, colorReset)
	_ = runCommand("psql", "-h", dbHost, "-p", dbPort, "-U", dbUser, "-c", fmt.Sprintf("DROP DATABASE IF EXISTS %s;", dbName))

	if err := runCommand("psql", "-h", dbHost, "-p", dbPort, "-U", dbUser, "-c", fmt.Sprintf("CREATE DATABASE %s;", dbName)); err != nil {
		fmt.Printf("%sError creating database: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}

	// Set environment variables for goose to avoid passing secrets in arguments
	os.Setenv("GOOSE_DRIVER", "postgres")
	os.Setenv("GOOSE_DBSTRING", dbURL)

	// Test UP migrations
	fmt.Printf("%sTesting UP migrations...%s\n", colorYellow, colorReset)
	if err := runCommandVerbose("goose", "-dir", "migrations", "up"); err != nil {
		fmt.Printf("%sError running UP migrations: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}

	// Verify UP
	versionUp, err := getGooseVersion()
	if err != nil {
		fmt.Printf("%sError getting goose version: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}
	// "goose: version 0" means no migrations applied? No, it means version 0.
	// We expect version > 0 or a timestamp.
	// Output is usually "goose: version 2023..."
	if strings.Contains(versionUp, "version 0") {
		fmt.Printf("%sError: UP migrations did not update version (version is 0)%s\n", colorRed, colorReset)
		os.Exit(1)
	}
	fmt.Printf("%s✓ UP migrations completed (Version: %s)%s\n", colorGreen, versionUp, colorReset)


	// Test DOWN migrations (all the way)
	fmt.Printf("%sTesting DOWN migrations (all)...%s\n", colorYellow, colorReset)
	if err := runCommandVerbose("goose", "-dir", "migrations", "down-to", "0"); err != nil {
		fmt.Printf("%sError running DOWN migrations: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}

	// Verify DOWN
	versionDown, err := getGooseVersion()
	if err != nil {
		fmt.Printf("%sError getting goose version: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}
	// We expect version 0
	// Output: "goose: version 0"
	if !strings.Contains(versionDown, "version 0") {
		fmt.Printf("%sError: DOWN migrations did not reset version (Version: %s)%s\n", colorRed, versionDown, colorReset)
		os.Exit(1)
	}
	fmt.Printf("%s✓ DOWN migrations completed (Version: %s)%s\n", colorGreen, versionDown, colorReset)


	// Test UP migrations again (idempotency)
	fmt.Printf("%sTesting UP migrations again (idempotency)...%s\n", colorYellow, colorReset)
	if err := runCommandVerbose("goose", "-dir", "migrations", "up"); err != nil {
		fmt.Printf("%sError running UP migrations again: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}

	// Verify Idempotency
	versionReUp, err := getGooseVersion()
	if err != nil {
		fmt.Printf("%sError getting goose version: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}
	if versionReUp != versionUp {
		fmt.Printf("%sError: Migration count/version mismatch (%s vs %s)%s\n", colorRed, versionUp, versionReUp, colorReset)
		os.Exit(1)
	}
	fmt.Printf("%s✓ UP migrations completed again (Version: %s)%s\n", colorGreen, versionReUp, colorReset)

	// Final verification
	fmt.Printf("%s✅ All migration tests passed!%s\n", colorGreen, colorReset)
}

func getGooseVersion() (string, error) {
	// getCommandOutput uses exec.Command which inherits env vars
	out, err := getCommandOutput("goose", "-dir", "migrations", "version")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}
