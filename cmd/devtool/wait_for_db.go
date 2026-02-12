package main

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type WaitForDBCommand struct{}

func (c *WaitForDBCommand) Name() string {
	return "wait-for-db"
}

func (c *WaitForDBCommand) Description() string {
	return "Wait for database to be ready (with retries)"
}

func (c *WaitForDBCommand) Run(args []string) error {
	PrintHeader("Waiting for database...")

	// Helper to get env with fallback
	getEnv := func(key, fallback string) string {
		if value, ok := os.LookupEnv(key); ok {
			return value
		}
		return fallback
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbUser := getEnv("DB_USER", "dev")
		dbPass := getEnv("DB_PASSWORD", "change_this_secure_password")
		dbHost := getEnv("DB_HOST", "localhost")
		dbPort := getEnv("DB_PORT", "5432")
		dbName := getEnv("DB_NAME", "app")

		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPass, dbHost, dbPort, dbName)
	}

	maxRetries := 30
	retryInterval := 2 * time.Second

	for i := 0; i < maxRetries; i++ {
		db, err := sql.Open("pgx", dbURL)
		if err == nil {
			err = db.Ping()
			if err == nil {
				db.Close()
				PrintSuccess("Database is ready")
				return nil
			}
			db.Close()
		}

		fmt.Printf("Database not ready (%d/%d): %v\n", i+1, maxRetries, err)
		time.Sleep(retryInterval)
	}

	return fmt.Errorf("database failed to become ready after %d attempts", maxRetries)
}
