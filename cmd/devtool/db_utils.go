package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

// RotateBackups keeps only the latest maxKeep files in dir that start with prefix.
func RotateBackups(dir, prefix string, maxKeep int) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var backups []os.DirEntry
	for _, f := range files {
		if !f.IsDir() && strings.HasPrefix(f.Name(), prefix) {
			backups = append(backups, f)
		}
	}

	if len(backups) <= maxKeep {
		return nil
	}

	// Sort by Name (which contains timestamp) descending, or use ModTime
	// Since our filenames are backup_env_YYYYMMDD_HHMMSS.sql, Name sort works.
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Name() > backups[j].Name()
	})

	for i := maxKeep; i < len(backups); i++ {
		path := filepath.Join(dir, backups[i].Name())
		if err := os.Remove(path); err != nil {
			PrintWarning("Failed to remove old backup %s: %v", path, err)
		} else {
			PrintInfo("Removed old backup: %s", path)
		}
	}

	return nil
}
