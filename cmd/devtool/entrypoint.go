package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type EntrypointCommand struct{}

func (c *EntrypointCommand) Name() string {
	return "entrypoint"
}

func (c *EntrypointCommand) Description() string {
	return "Container entrypoint (wait-for-db, backup, migrate, exec)"
}

func (c *EntrypointCommand) Run(args []string) error {
	// Set default DB_HOST to "db" if not set, mirroring previous script behavior
	if os.Getenv("DB_HOST") == "" {
		if err := os.Setenv("DB_HOST", "db"); err != nil {
			return fmt.Errorf("failed to set default DB_HOST: %w", err)
		}
	}

	// 1. Wait for DB
	waitCmd := &WaitForDBCommand{}
	if err := waitCmd.Run(nil); err != nil {
		return fmt.Errorf("wait-for-db failed: %w", err)
	}

	// 2. Backup if needed
	environment := os.Getenv("ENVIRONMENT")
	createBackup := os.Getenv("CREATE_BACKUP")
	if environment == "production" || createBackup == "true" {
		PrintHeader("Creating pre-migration backup...")

		// Check if pg_dump is available
		if _, err := exec.LookPath("pg_dump"); err == nil {
			timestamp := time.Now().Format("20060102_150405")
			backupFile := fmt.Sprintf("/tmp/backup_%s.sql", timestamp)

			dbUser := os.Getenv("DB_USER")
			dbName := os.Getenv("DB_NAME")
			dbHost := os.Getenv("DB_HOST")

			f, err := os.Create(backupFile)
			if err != nil {
				PrintWarning("Could not create backup file: %v", err)
			} else {
				// nolint:gosec
				cmd := exec.Command("pg_dump", "-h", dbHost, "-U", dbUser, "-d", dbName)
				cmd.Stdout = f
				cmd.Stderr = os.Stderr

				if err := cmd.Run(); err != nil {
					PrintWarning("Backup failed: %v", err)
					f.Close()
					// Don't fail the entrypoint, just warn, as per script behavior
				} else {
					f.Close()
					PrintSuccess("Backup created: %s", backupFile)
				}
			}
		} else {
			PrintWarning("pg_dump not found, skipping backup")
		}
	}

	// 3. Migrate
	PrintHeader("Running migrations...")
	migrateCmd := &MigrateCommand{}

	maxRetries := 3
	var err error
	for i := 0; i < maxRetries; i++ {
		err = migrateCmd.Run([]string{"up"})
		if err == nil {
			break
		}
		PrintWarning("Migration attempt %d failed: %v", i+1, err)
		if i < maxRetries-1 {
			PrintInfo("Retrying in 5 seconds...")
			time.Sleep(5 * time.Second)
		}
	}
	if err != nil {
		return fmt.Errorf("migrations failed after %d attempts: %w", maxRetries, err)
	}
	PrintSuccess("Migrations completed successfully")

	// 4. Exec
	// Handle optional "--" separator
	execArgs := args
	if len(execArgs) > 0 && execArgs[0] == "--" {
		execArgs = execArgs[1:]
	}

	if len(execArgs) == 0 {
		return fmt.Errorf("no command to execute")
	}

	PrintHeader("Starting application...")
	cmdPath, err := exec.LookPath(execArgs[0])
	if err != nil {
		return fmt.Errorf("executable not found: %w", err)
	}

	// syscall.Exec replaces the current process
	if err := syscall.Exec(cmdPath, execArgs, os.Environ()); err != nil {
		return fmt.Errorf("exec failed: %w", err)
	}

	return nil // Should not be reached
}
