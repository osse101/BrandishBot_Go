package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"github.com/osse101/BrandishBot_Go/internal/database/schema"
)

func main() {
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

	// 4. Execute schema
	fmt.Println("Applying database schema...")
	_, err = targetConn.Exec(context.Background(), schema.SchemaSQL)
	if err != nil {
		log.Fatalf("Failed to execute schema: %v", err)
	}

	fmt.Println("Schema applied successfully.")
}
