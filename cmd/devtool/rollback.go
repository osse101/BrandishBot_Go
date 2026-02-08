package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

type RollbackCommand struct{}

func (c *RollbackCommand) Name() string {
	return "rollback"
}

func (c *RollbackCommand) Description() string {
	return "Rollback to a previous version"
}

func (c *RollbackCommand) Run(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: devtool rollback <environment> [version]")
	}

	env := args[0]
	version := ""
	if len(args) > 1 {
		version = args[1]
	}

	if env != "staging" && env != "production" {
		return fmt.Errorf("environment must be 'staging' or 'production'")
	}

	composeFile := "docker-compose.staging.yml"
	if env == "production" {
		composeFile = "docker-compose.production.yml"
	}

	PrintHeader(fmt.Sprintf("BrandishBot Rollback (%s)", env))

	if version == "" {
		PrintInfo("Available Docker images (last 10):")
		// Need to execute sh -c for pipe
		cmd := exec.Command("sh", "-c", "docker images brandishbot --format \"table {{.Tag}}\t{{.CreatedAt}}\" | head -n 11")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Run()

		fmt.Println()
		fmt.Print("Enter version to rollback to (or 'cancel' to abort): ")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			version = strings.TrimSpace(scanner.Text())
		}
		if version == "" || version == "cancel" {
			return fmt.Errorf("rollback cancelled")
		}
	}

	// Verify image exists
	out, err := getCommandOutput("docker", "images", fmt.Sprintf("brandishbot:%s", version), "--format", "{{.Tag}}")
	if err != nil || out == "" {
		PrintError("Docker image brandishbot:%s not found", version)
		return fmt.Errorf("image not found")
	}

	if env == "production" {
		PrintWarning("You are about to rollback PRODUCTION to version %s", version)
		fmt.Print("Type 'yes' to continue: ")
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "yes" {
			return fmt.Errorf("rollback cancelled")
		}
	}

	// Step 1: Stop current containers
	PrintInfo("Step 1/4: Stopping current containers")
	runCommandVerbose("docker", "compose", "-f", composeFile, "stop", "app", "discord")

	// Step 2: Rollback
	PrintInfo("Step 2/4: Rolling back to version %s", version)
	os.Setenv("DOCKER_IMAGE_TAG", version)
	if err := runCommandVerbose("docker", "compose", "-f", composeFile, "up", "-d", "--no-deps", "app", "discord"); err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	// Step 3: Wait for health checks
	PrintInfo("Step 3/4: Waiting for health checks (max 60s)")
	if err := waitForHealth(env, 60*time.Second); err != nil {
		PrintError("Health check failed after rollback")
		PrintInfo("Check logs: docker compose -f %s logs app", composeFile)
		return err
	}
	PrintSuccess("Health checks passed")

	// Step 4: Database rollback option
	PrintInfo("Step 4/4: Database rollback")
	PrintWarning("Do you need to restore the database from a backup?")
	PrintInfo("Available backups:")
	cmd := exec.Command("sh", "-c", fmt.Sprintf("ls -lth backups/backup_%s_*.sql 2>/dev/null | head -n 5 || echo 'No backups found'", env))
	cmd.Stdout = os.Stdout
	_ = cmd.Run()

	fmt.Println()
	fmt.Print("Enter backup filename to restore (or press Enter to skip): ")
	scanner := bufio.NewScanner(os.Stdin)
	backupFile := ""
	if scanner.Scan() {
		backupFile = strings.TrimSpace(scanner.Text())
	}

	if backupFile != "" {
		if _, err := os.Stat(backupFile); os.IsNotExist(err) {
			PrintError("Backup file not found: %s", backupFile)
		} else {
			PrintWarning("This will overwrite the current database!")
			fmt.Printf("Type 'yes' to restore database from %s: ", backupFile)
			var confirmDB string
			fmt.Scanln(&confirmDB)
			if confirmDB == "yes" {
				PrintInfo("Restoring database from %s...", backupFile)
				dbContainerID, _ := getCommandOutput("docker", "compose", "-f", composeFile, "ps", "-q", "db")
				if dbContainerID == "" {
					PrintError("Database container not running")
				} else {
					// Use sh -c to pipe input file
					restoreCmd := exec.Command("sh", "-c", fmt.Sprintf("docker exec -i %s psql -U brandishbot -d brandishbot < %s", dbContainerID, backupFile))
					restoreCmd.Stdout = os.Stdout
					restoreCmd.Stderr = os.Stderr
					if err := restoreCmd.Run(); err != nil {
						PrintError("Database restore failed: %v", err)
					} else {
						PrintSuccess("Database restored")
					}
				}
			} else {
				PrintInfo("Database restore cancelled")
			}
		}
	} else {
		PrintInfo("Skipping database restore")
	}

	PrintSuccess("=== Rollback Complete ===")
	return nil
}
