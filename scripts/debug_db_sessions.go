package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close(ctx)

	rows, err := conn.Query(ctx, `
		SELECT s.id, s.status, s.started_at, 
		       (SELECT COUNT(*) FROM progression_voting_options o WHERE o.session_id = s.id) as option_count 
		FROM progression_voting_sessions s 
		ORDER BY s.started_at DESC LIMIT 20;
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Printf("%-5s | %-10s | %-30s | %-12s\n", "ID", "Status", "Started At", "Option Count")
	fmt.Println("-------------------------------------------------------------------------")
	for rows.Next() {
		var id int
		var status string
		var startedAt interface{}
		var count int
		if err := rows.Scan(&id, &status, &startedAt, &count); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%-5d | %-10s | %-30v | %-12d\n", id, status, startedAt, count)
	}
}
