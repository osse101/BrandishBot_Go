package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/osse101/BrandishBot_Go/internal/database"
	"github.com/osse101/BrandishBot_Go/internal/database/postgres"
	"github.com/osse101/BrandishBot_Go/internal/server"
	"github.com/osse101/BrandishBot_Go/internal/user"
)

func main() {
	// Setup logging
	logFile, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()

	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)

	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using default/environment variables")
	}

	// Construct connection string
	connString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	dbPool, err := database.NewPool(connString)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	userRepo := postgres.NewUserRepository(dbPool)
	userService := user.NewService(userRepo)
	srv := server.NewServer(8080, userService)

	if err := srv.Start(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
