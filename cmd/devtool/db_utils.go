package main

import (
	"fmt"
	"os"
)

// GetDBURL retrieves the database URL from environment variables or constructs a default one.
func GetDBURL() string {
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

	return dbURL
}
