package main

import (
	"database/sql"
	"fmt"
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

	dbURL := GetDBURL()

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
