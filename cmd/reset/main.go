package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"

	"github.com/osse101/BrandishBot_Go/internal/database"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	dbName := os.Getenv("DB_NAME")

	// Connect to PostgreSQL server (postgres database to manage other databases)
	serverConnString := fmt.Sprintf("postgres://%s:%s@%s:%s/postgres?sslmode=disable",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
	)

	serverPool, err := database.NewPool(serverConnString, 10, 30*time.Minute, time.Hour)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL server: %v", err)
	}
	defer serverPool.Close()

	ctx := context.Background()

	// Terminate existing connections to the database
	log.Printf("Terminating existing connections to database %s...\n", dbName)
	_, err = serverPool.Exec(ctx, fmt.Sprintf(`
		SELECT pg_terminate_backend(pg_stat_activity.pid)
		FROM pg_stat_activity
		WHERE pg_stat_activity.datname = '%s'
		AND pid <> pg_backend_pid()
	`, dbName))
	if err != nil {
		log.Printf("Warning: Failed to terminate connections: %v\n", err)
	}

	// Drop database if exists
	log.Printf("Dropping database %s if it exists...\n", dbName)
	_, err = serverPool.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
	if err != nil {
		log.Fatalf("Failed to drop database: %v", err)
	}
	log.Printf("Database %s dropped successfully.\n", dbName)

	// Create database
	log.Printf("Creating database %s...\n", dbName)
	_, err = serverPool.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", dbName))
	if err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}
	log.Printf("Database %s created successfully.\n", dbName)

	log.Println("\n✅ Database reset complete!")
	log.Println("Next step: Run 'make migrate-up' to apply migrations")
}
