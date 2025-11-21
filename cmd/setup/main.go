package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	// 1. Connect to default 'postgres' database to create the new database
	defaultConnString := fmt.Sprintf("postgres://%s:%s@%s:%s/postgres?sslmode=disable", user, password, host, port)
	conn, err := pgx.Connect(context.Background(), defaultConnString)
	if err != nil {
		log.Fatalf("Unable to connect to postgres database: %v", err)
	}
	defer conn.Close(context.Background())

	// 2. Check if database exists
	var exists bool
	err = conn.QueryRow(context.Background(), "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", dbname).Scan(&exists)
	if err != nil {
		log.Fatalf("Failed to check if database exists: %v", err)
	}

	if !exists {
		fmt.Printf("Creating database %s...\n", dbname)
		_, err = conn.Exec(context.Background(), fmt.Sprintf("CREATE DATABASE %s", dbname))
		if err != nil {
			log.Fatalf("Failed to create database: %v", err)
		}
		fmt.Println("Database created successfully.")
	} else {
		fmt.Printf("Database %s already exists.\n", dbname)
	}

	// Close connection to postgres db
	conn.Close(context.Background())

	// 3. Connect to the new database to run migrations
	targetConnString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, port, dbname)
	targetConn, err := pgx.Connect(context.Background(), targetConnString)
	if err != nil {
		log.Fatalf("Unable to connect to %s database: %v", dbname, err)
	}
	defer targetConn.Close(context.Background())

	// 4. Read migration file
	migrationPath := filepath.Join("migrations", "0001_initial_schema.up.sql")
	migrationSQL, err := os.ReadFile(migrationPath)
	if err != nil {
		log.Fatalf("Failed to read migration file: %v", err)
	}

	// 5. Execute migration
	fmt.Println("Running migration...")
	_, err = targetConn.Exec(context.Background(), string(migrationSQL))
	if err != nil {
		log.Fatalf("Failed to execute migration: %v", err)
	}

	fmt.Println("Migration completed successfully.")
}
