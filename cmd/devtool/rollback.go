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

	if env != envStaging && env != envProduction {
		return fmt.Errorf("environment must be '%s' or '%s'", envStaging, envProduction)
	}

	composeFile := "docker-compose.staging.yml"
	if env == envProduction {
		composeFile = "docker-compose.production.yml"
	}

	PrintHeader(fmt.Sprintf("BrandishBot Rollback (%s)", env))

	// Prompt for version if not provided
	var err error
	if version == "" {
		version, err = c.promptForVersion()
		if err != nil {
			return err
		}
	}

	// Verify image exists
	if err := c.verifyImageExists(version); err != nil {
		return err
	}

	// Confirm production rollback
	if env == envProduction {
		if err := c.confirmProductionRollback(version); err != nil {
			return err
		}
	}

	// Execute rollback steps
	if err := c.executeRollback(env, version, composeFile); err != nil {
		return err
	}

	// Optionally restore database
	if err := c.handleDatabaseRestore(env, composeFile); err != nil {
		return err
	}

	PrintSuccess("=== Rollback Complete ===")
	return nil
}

func (c *RollbackCommand) promptForVersion() (string, error) {
	PrintInfo("Available Docker images (last 10):")

	// Validate appName constant
	if appName == "" || strings.ContainsAny(appName, ";|&$`\"'") {
		return "", fmt.Errorf("invalid app name constant")
	}

	//nolint:gosec // G204: appName is validated above
	cmd := exec.Command("sh", "-c", fmt.Sprintf("docker images %s --format \"table {{.Tag}}\t{{.CreatedAt}}\" | head -n 11", appName))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		PrintWarning("Failed to list images: %v", err)
	}

	fmt.Println()
	fmt.Print("Enter version to rollback to (or 'cancel' to abort): ")
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		version := strings.TrimSpace(scanner.Text())
		if version == "" || version == "cancel" {
			return "", fmt.Errorf("rollback cancelled")
		}
		return version, nil
	}
	return "", fmt.Errorf("failed to read input: %w", scanner.Err())
}

func (c *RollbackCommand) verifyImageExists(version string) error {
	out, err := getCommandOutput("docker", "images", fmt.Sprintf("%s:%s", appName, version), "--format", "{{.Tag}}")
	if err != nil || out == "" {
		PrintError("Docker image %s:%s not found", appName, version)
		return fmt.Errorf("image not found")
	}
	return nil
}

func (c *RollbackCommand) confirmProductionRollback(version string) error {
	PrintWarning("You are about to rollback PRODUCTION to version %s", version)
	fmt.Print("Type 'yes' to continue: ")
	var confirm string
	if _, err := fmt.Scanln(&confirm); err != nil {
		return fmt.Errorf("failed to read confirmation: %w", err)
	}
	if confirm != confirmYes {
		return fmt.Errorf("rollback cancelled")
	}
	return nil
}

func (c *RollbackCommand) executeRollback(env, version, composeFile string) error {
	// Step 1: Stop current containers
	PrintInfo("Step 1/3: Stopping current containers")
	if err := runCommandVerbose("docker", "compose", "-f", composeFile, "stop", "app", "discord"); err != nil {
		PrintWarning("Failed to stop containers cleanly: %v", err)
	}

	// Step 2: Rollback
	PrintInfo("Step 2/3: Rolling back to version %s", version)
	os.Setenv("DOCKER_IMAGE_TAG", version)
	if err := runCommandVerbose("docker", "compose", "-f", composeFile, "up", "-d", "--no-deps", "app", "discord"); err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	// Step 3: Wait for health checks
	PrintInfo("Step 3/3: Waiting for health checks (max 60s)")
	if err := waitForHealth(env, 60*time.Second); err != nil {
		PrintError("Health check failed after rollback")
		PrintInfo("Check logs: docker compose -f %s logs app", composeFile)
		return err
	}
	PrintSuccess("Health checks passed")
	return nil
}

func (c *RollbackCommand) handleDatabaseRestore(env, composeFile string) error {
	PrintInfo("Database rollback")
	PrintWarning("Do you need to restore the database from a backup?")
	PrintInfo("Available backups:")

	// Validate env (should be staging or production)
	if env != envStaging && env != envProduction {
		return fmt.Errorf("invalid environment for database restore")
	}

	//nolint:gosec // G204: env is validated above against constants
	cmd := exec.Command("sh", "-c", fmt.Sprintf("ls -lth backups/backup_%s_*.sql 2>/dev/null | head -n 5 || echo 'No backups found'", env))
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		PrintWarning("Failed to list backups: %v", err)
	}

	fmt.Println()
	fmt.Print("Enter backup filename to restore (or press Enter to skip): ")
	scanner := bufio.NewScanner(os.Stdin)
	backupFile := ""
	if scanner.Scan() {
		backupFile = strings.TrimSpace(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		PrintWarning("Failed to read backup filename: %v", err)
	}

	if backupFile != "" {
		return c.restoreDatabase(backupFile, composeFile)
	}
	PrintInfo("Skipping database restore")
	return nil
}

func (c *RollbackCommand) restoreDatabase(backupFile, composeFile string) error {
	// Validate backup file path to prevent path traversal
	if backupFile == "" {
		PrintError("Backup file path is empty")
		return nil
	}
	if strings.Contains(backupFile, "..") || strings.ContainsAny(backupFile, ";|&$`\"'") {
		PrintError("Invalid characters in backup file path")
		return nil
	}
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		PrintError("Backup file not found: %s", backupFile)
		return nil
	}

	PrintWarning("This will overwrite the current database!")
	fmt.Printf("Type 'yes' to restore database from %s: ", backupFile)
	var confirmDB string
	if _, err := fmt.Scanln(&confirmDB); err != nil {
		PrintWarning("Failed to read confirmation: %v", err)
		return nil
	}

	if confirmDB != confirmYes {
		PrintInfo("Database restore cancelled")
		return nil
	}

	PrintInfo("Restoring database from %s...", backupFile)
	dbContainerID, _ := getCommandOutput("docker", "compose", "-f", composeFile, "ps", "-q", "db")
	if dbContainerID == "" {
		PrintError("Database container not running")
		return nil
	}

	// Validate dbContainerID and appName
	if dbContainerID == "" || strings.ContainsAny(dbContainerID, ";|&$`\"'") {
		PrintError("Invalid database container ID")
		return nil
	}
	if appName == "" || strings.ContainsAny(appName, ";|&$`\"'") {
		PrintError("Invalid app name constant")
		return nil
	}

	//nolint:gosec // G204: all parameters are validated above
	restoreCmd := exec.Command("sh", "-c", fmt.Sprintf("docker exec -i %s psql -U %s -d %s < %s", dbContainerID, appName, appName, backupFile))
	restoreCmd.Stdout = os.Stdout
	restoreCmd.Stderr = os.Stderr
	if err := restoreCmd.Run(); err != nil {
		PrintError("Database restore failed: %v", err)
		return err
	}
	PrintSuccess("Database restored")
	return nil
}
