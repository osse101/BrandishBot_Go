package main

import (
	"fmt"
	"os"
)

type MigrateCommand struct{}

func (c *MigrateCommand) Name() string {
	return "migrate"
}

func (c *MigrateCommand) Description() string {
	return "Manage database migrations (up, down, status, create)"
}

func (c *MigrateCommand) Run(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("subcommand required: up, down, status, create")
	}
	subcmd := args[0]

	// Goose command and arguments
	gooseCmd := "go"
	gooseArgs := []string{"run", "github.com/pressly/goose/v3/cmd/goose", "-dir", "migrations"}

	// Handle create command (no DB connection needed)
	if subcmd == "create" {
		if len(args) < 2 {
			return fmt.Errorf("migration name required for create")
		}

		// Add create subcommand
		gooseArgs = append(gooseArgs, "create")

		// Add name
		gooseArgs = append(gooseArgs, args[1])

		// Add type (default to sql if not provided, though devtool args[2] might be type if passed)
		// But usually we just pass name.
		// Makefile: @$(GOOSE) -dir migrations create $(NAME) sql
		// So we default to sql.
		migrationType := "sql"
		if len(args) > 2 {
			migrationType = args[2]
		}
		gooseArgs = append(gooseArgs, migrationType)

		return runCommandVerbose(gooseCmd, gooseArgs...)
	}

	// For other commands, we need DB connection
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

	// Add driver and dbstring
	gooseArgs = append(gooseArgs, "postgres", dbURL)

	// Add subcommand
	gooseArgs = append(gooseArgs, subcmd)

	// Add any extra args (e.g. version for up-to/down-to)
	if len(args) > 1 {
		gooseArgs = append(gooseArgs, args[1:]...)
	}

	return runCommandVerbose(gooseCmd, gooseArgs...)
}
