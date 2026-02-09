package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

type DeployCommand struct{}

func (c *DeployCommand) Name() string {
	return "deploy"
}

func (c *DeployCommand) Description() string {
	return "Deploy the application (local build or remote pull)"
}

func (c *DeployCommand) Run(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: devtool deploy <environment> <version> [--remote]")
	}

	env := args[0]
	version := args[1]
	remote := false

	if len(args) > 2 && args[2] == "--remote" {
		remote = true
	}

	if env != envStaging && env != envProduction {
		return fmt.Errorf("environment must be '%s' or '%s'", envStaging, envProduction)
	}

	PrintHeader(fmt.Sprintf("BrandishBot Deployment (%s)", env))
	PrintInfo("Environment: %s", env)
	PrintInfo("Version: %s", version)
	PrintInfo("Mode: %s", map[bool]string{true: "Remote (Pull)", false: "Local (Build)"}[remote])

	if env == envProduction {
		PrintWarning("You are about to deploy to PRODUCTION")
		fmt.Print("Type 'yes' to continue: ")
		var confirm string
		if _, err := fmt.Scanln(&confirm); err != nil {
			return fmt.Errorf("failed to read confirmation: %w", err)
		}
		if confirm != confirmYes {
			return fmt.Errorf("deployment cancelled")
		}
	}

	composeFile := "docker-compose.staging.yml"
	if env == envProduction {
		composeFile = "docker-compose.production.yml"
	}

	if remote {
		return c.deployRemote(env, version, composeFile)
	}
	return c.deployLocal(env, version, composeFile)
}

func (c *DeployCommand) deployLocal(env, version, composeFile string) error {
	// Step 1: Pre-deployment health check
	PrintInfo("Step 1/7: Pre-deployment health check")
	if err := checkHealth(env); err != nil {
		PrintWarning("Pre-deployment health check failed: %v", err)
	} else {
		PrintSuccess("Current deployment is healthy")
	}

	// Step 2: Database backup
	PrintInfo("Step 2/7: Creating database backup")
	if err := backupDatabase(env, composeFile); err != nil {
		PrintWarning("Database backup failed or skipped: %v", err)
	}

	// Step 3: Build Docker image
	PrintInfo("Step 3/7: Building Docker image")
	imageName := os.Getenv("DOCKER_IMAGE_NAME")
	if imageName == "" {
		imageName = appName
	}
	dockerUser := os.Getenv("DOCKER_USER")
	fullImageName := imageName
	if dockerUser != "" {
		fullImageName = fmt.Sprintf("%s/%s", dockerUser, imageName)
	}

	buildArgs := []string{
		"build",
		"--build-arg", fmt.Sprintf("VERSION=%s", version),
		"-t", fmt.Sprintf("%s:%s", fullImageName, version),
		"-t", fmt.Sprintf("%s:latest-%s", fullImageName, env),
		"-f", "Dockerfile",
		".",
	}
	if err := runCommandVerbose("docker", buildArgs...); err != nil {
		return fmt.Errorf("docker build failed: %w", err)
	}
	PrintSuccess("Docker image built: %s:%s", fullImageName, version)

	// Step 4: Deploy new containers
	PrintInfo("Step 4/7: Deploying new containers")
	os.Setenv("DOCKER_IMAGE_TAG", version)
	if err := runCommandVerbose("docker", "compose", "-f", composeFile, "up", "-d", "app", "discord"); err != nil {
		PrintError("Deployment failed, attempting rollback...")
		_ = runCommand("docker", "compose", "-f", composeFile, "up", "-d", "--no-deps", "app", "discord")
		return fmt.Errorf("deployment failed: %w", err)
	}
	PrintSuccess("Containers deployed")

	// Step 5: Wait for health checks
	PrintInfo("Step 5/7: Waiting for health checks (max 60s)")
	if err := waitForHealth(env, 60*time.Second); err != nil {
		PrintError("Health check failed after deployment")
		PrintInfo("Check logs: docker compose -f %s logs app", composeFile)
		return err
	}
	PrintSuccess("Health checks passed")

	// Step 6: Smoke tests
	PrintInfo("Step 6/7: Running smoke tests")
	port := "8080"
	if env == "staging" {
		port = "8081"
	}
	if err := runSmokeTests(port); err != nil {
		return err
	}

	// Step 7: Cleanup old images
	PrintInfo("Step 7/7: Cleaning up old Docker images")
	cleanupOldImages(appName)

	PrintSuccess("=== Deployment Complete ===")
	return nil
}

func (c *DeployCommand) deployRemote(env, version, composeFile string) error {
	// 1. Docker Login
	if err := runCommand("docker", "system", "info"); err != nil || !strings.Contains(func() string { out, _ := getCommandOutput("docker", "system", "info"); return out }(), "Username") {
		PrintWarning("Not logged in. Attempting docker login...")
		if err := runCommandVerbose("docker", "login"); err != nil {
			return fmt.Errorf("docker login failed: %w", err)
		}
	}

	// 2. Pull images
	PrintInfo("Pulling images...")
	os.Setenv("DOCKER_IMAGE_TAG", version)
	if err := runCommandVerbose("docker", "compose", "-f", composeFile, "pull", "app", "discord"); err != nil {
		return fmt.Errorf("failed to pull images: %w", err)
	}

	// 3. Restart services
	PrintInfo("Starting services...")
	if err := runCommand("docker", "compose", "-f", composeFile, "up", "-d", "db"); err != nil {
		return fmt.Errorf("failed to start database: %w", err)
	}
	time.Sleep(2 * time.Second)
	if err := runCommandVerbose("docker", "compose", "-f", composeFile, "up", "-d", "app", "discord"); err != nil {
		return fmt.Errorf("failed to start services: %w", err)
	}
	PrintSuccess("Services started")

	// 4. Prune old images
	PrintInfo("Cleaning up old images...")
	if err := runCommand("docker", "image", "prune", "-f"); err != nil {
		PrintWarning("Failed to prune old images: %v", err)
	}

	// 5. Health Check
	PrintInfo("Running health checks...")
	if err := checkHealth(env); err != nil {
		PrintWarning("Health check failed: %v", err)
	} else {
		PrintSuccess("Health check passed")
	}

	// 6. Release Notes
	c.announceRelease(version)

	PrintSuccess("=== Remote Deployment Complete ===")
	return nil
}

func (c *DeployCommand) announceRelease(version string) {
	PrintInfo("Generating release notes for version %s...", version)

	lastTag, err := getCommandOutput("git", "describe", "--tags", "--abbrev=0")
	rangeSpec := "HEAD~5..HEAD"
	title := fmt.Sprintf("Deployment Update (%s)", version)

	if err == nil && lastTag != "" {
		rangeSpec = fmt.Sprintf("%s..HEAD", lastTag)
		title = fmt.Sprintf("Deployment Update (%s -> HEAD)", lastTag)
	}

	notes, _ := getCommandOutput("git", "log", "--pretty=format:• %s (%an)", rangeSpec, "-n", "20")
	if notes == "" {
		notes = "No new commits in this deployment."
	}

	payload := map[string]interface{}{
		"title":       title,
		"description": notes,
		"color":       65280,
	}
	jsonPayload, _ := json.Marshal(payload)

	discordURL := "http://localhost:8082/admin/announce"
	resp, err := http.Post(discordURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		PrintWarning("Failed to send release notes: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		PrintSuccess("Release notes sent successfully")
	} else {
		PrintWarning("Failed to send release notes (Status: %d)", resp.StatusCode)
	}
}

func checkHealth(env string) error {
	port := "8080"
	if env == envStaging {
		port = "8081"
	}
	url := fmt.Sprintf("http://localhost:%s/healthz", port)

	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		// Try 127.0.0.1
		url = fmt.Sprintf("http://127.0.0.1:%s/healthz", port)
		resp, err = client.Get(url)
		if err != nil {
			return err
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("status code %d", resp.StatusCode)
	}
	return nil
}

func waitForHealth(env string, timeout time.Duration) error {
	start := time.Now()
	for time.Since(start) < timeout {
		if err := checkHealth(env); err == nil {
			return nil
		}
		time.Sleep(2 * time.Second)
		fmt.Print(".")
	}
	fmt.Println()
	return fmt.Errorf("timeout waiting for health check")
}

func backupDatabase(env, composeFile string) error {
	// Check if DB is up
	out, err := getCommandOutput("docker", "compose", "-f", composeFile, "ps", "-q", "db")
	if err != nil || out == "" {
		return fmt.Errorf("database container not found or not running")
	}

	if err := os.MkdirAll("backups", 0755); err != nil {
		return err
	}

	filename := fmt.Sprintf("backups/backup_%s_%s.sql", env, time.Now().Format("20060102_150405"))
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = appName
	}
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = appName
	}

	cmd := exec.Command("docker", "exec", out, "pg_dump", "-U", dbUser, "-d", dbName)
	outfile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer outfile.Close()
	cmd.Stdout = outfile

	if err := cmd.Run(); err != nil {
		return err
	}
	PrintSuccess("Database backup created: %s", filename)
	return nil
}

func runSmokeTests(port string) error {
	urls := []string{
		fmt.Sprintf("http://localhost:%s/healthz", port),
		fmt.Sprintf("http://localhost:%s/progression/tree", port),
	}

	for _, url := range urls {
		resp, err := http.Get(url)
		if err != nil {
			PrintWarning("Smoke test failed for %s: %v", url, err)
			continue
		}
		resp.Body.Close()
		if resp.StatusCode == 200 {
			PrintSuccess("✓ %s responding", url)
		} else {
			PrintWarning("✗ %s returned %d", url, resp.StatusCode)
		}
	}
	return nil
}

func cleanupOldImages(imageName string) {
	// This is a bit complex to do cross-platform with just exec,
	// simpler to shell out or use docker Go SDK.
	// For now, mirroring the shell script logic roughly.
	cmd := exec.Command("sh", "-c", fmt.Sprintf("docker images \"%s\" --format \"{{.Tag}}\" | grep -v \"latest\" | tail -n +6 | xargs -r -I {} docker rmi \"%s:{}\"", imageName, imageName))
	_ = cmd.Run()
}
