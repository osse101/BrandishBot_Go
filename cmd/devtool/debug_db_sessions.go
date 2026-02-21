package main

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type DebugDBSessionsCommand struct{}

func (c *DebugDBSessionsCommand) Name() string {
	return "debug-db-sessions"
}

func (c *DebugDBSessionsCommand) Description() string {
	return "Debug progression voting sessions (last 20)"
}

func (c *DebugDBSessionsCommand) Run(args []string) error {
	dbURL := GetDBURL()

	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to db: %w", err)
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT s.id, s.status, s.started_at, 
		       (SELECT COUNT(*) FROM progression_voting_options o WHERE o.session_id = s.id) as option_count 
		FROM progression_voting_sessions s 
		ORDER BY s.started_at DESC LIMIT 20;
	`)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	fmt.Printf("%-5s | %-10s | %-30s | %-12s\n", "ID", "Status", "Started At", "Option Count")
	fmt.Println("-------------------------------------------------------------------------")

	for rows.Next() {
		var id int
		var status string
		var startedAt any
		var count int
		if err := rows.Scan(&id, &status, &startedAt, &count); err != nil {
			return fmt.Errorf("scan failed: %w", err)
		}
		fmt.Printf("%-5d | %-10s | %-30v | %-12d\n", id, status, startedAt, count)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows iteration failed: %w", err)
	}

	return nil
}
