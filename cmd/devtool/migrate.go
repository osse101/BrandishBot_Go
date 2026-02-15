package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
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

	// Set the migration directory
	migrationDir := "migrations"
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	// Handle create command (no DB connection needed)
	if subcmd == "create" {
		if len(args) < 2 {
			return fmt.Errorf("migration name required for create")
		}

		migrationName := args[1]
		migrationType := "sql"
		if len(args) > 2 {
			migrationType = args[2]
		}

		return goose.Create(nil, migrationDir, migrationName, migrationType)
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

	// Open connection to DB
	// We use "pgx" driver name which is registered by pgx/v5/stdlib
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Execute command
	switch subcmd {
	case "up":
		return goose.Up(db, migrationDir)
	case "down":
		return goose.Down(db, migrationDir)
	case "status":
		return goose.Status(db, migrationDir)
	default:
		return fmt.Errorf("unknown subcommand: %s", subcmd)
	}
}
