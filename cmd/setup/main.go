package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/joho/godotenv"
	"github.com/osse101/BrandishBot_Go/internal/database"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Construct connection string
	connString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	// Connect to PostgreSQL server (without database name to create it if needed)
	serverConnString := fmt.Sprintf("postgres://%s:%s@%s:%s/postgres?sslmode=disable",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
	)

	serverPool, err := database.NewPool(serverConnString)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL server: %v", err)
	}
	defer serverPool.Close()

	// Create database if it doesn't exist
	dbName := os.Getenv("DB_NAME")
	ctx := context.Background()

	var exists bool
	err = serverPool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", dbName).Scan(&exists)
	if err != nil {
		log.Fatalf("Failed to check if database exists: %v", err)
	}

	if !exists {
		_, err = serverPool.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", dbName))
		if err != nil {
			log.Fatalf("Failed to create database: %v", err)
		}
		log.Printf("Database %s created successfully.\n", dbName)
	} else {
		log.Printf("Database %s already exists.\n", dbName)
	}

	// Close server connection
	serverPool.Close()

	// Connect to the specific database
	dbPool, err := database.NewPool(connString)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	// Read and apply migrations
	log.Println("Applying database migrations...")
	
	migrationsDir := "migrations"
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		log.Fatalf("Failed to read migrations directory: %v", err)
	}

	// Filter and sort .up.sql files
	var upFiles []string
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".sql" {
			if len(file.Name()) > 7 && file.Name()[len(file.Name())-7:] == ".up.sql" {
				upFiles = append(upFiles, file.Name())
			}
		}
	}
	sort.Strings(upFiles)

	// Apply each migration
	for _, filename := range upFiles {
		log.Printf("Applying migration: %s\n", filename)
		
		filePath := filepath.Join(migrationsDir, filename)
		content, err := os.ReadFile(filePath)
		if err != nil {
			log.Fatalf("Failed to read migration file %s: %v", filename, err)
		}

		_, err = dbPool.Exec(ctx, string(content))
		if err != nil {
			log.Fatalf("Failed to apply migration %s: %v", filename, err)
		}
	}

	log.Println("All migrations applied successfully!")
}
