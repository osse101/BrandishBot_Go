package main

import (
	"bytes"
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
	//nolint:forbidigo
	if err := runCommandVerbose("docker", buildArgs...); err != nil { // #nosec G204
		return fmt.Errorf("docker build failed: %w", err)
	}
	PrintSuccess("Docker image built: %s:%s", fullImageName, version)

	// Step 4: Deploy new containers
	PrintInfo("Step 4/7: Deploying new containers")
	os.Setenv("DOCKER_IMAGE_TAG", version)
	//nolint:forbidigo
	if err := runCommandVerbose("docker", "compose", "-f", composeFile, "up", "-d", "app", "discord"); err != nil { // #nosec G204
		PrintError("Deployment failed, attempting rollback...")
		//nolint:forbidigo
		_ = runCommand("docker", "compose", "-f", composeFile, "up", "-d", "--no-deps", "app", "discord") // #nosec G204
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
	//nolint:forbidigo
	isLoggedIn := func() bool {
		//nolint:forbidigo
		out, _ := getCommandOutput("docker", "system", "info")
		return strings.Contains(out, "Username")
	}
	//nolint:forbidigo
	if err := runCommand("docker", "system", "info"); err != nil || !isLoggedIn() {
		PrintWarning("Not logged in. Attempting docker login...")
		//nolint:forbidigo
		if err := runCommandVerbose("docker", "login"); err != nil {
			return fmt.Errorf("docker login failed: %w", err)
		}
	}

	// 2. Pull images
	PrintInfo("Pulling images...")
	os.Setenv("DOCKER_IMAGE_TAG", version)
	//nolint:forbidigo
	if err := runCommandVerbose("docker", "compose", "-f", composeFile, "pull", "app", "discord"); err != nil { // #nosec G204
		return fmt.Errorf("failed to pull images: %w", err)
	}

	// 3. Restart services
	PrintInfo("Starting services...")
	//nolint:forbidigo
	if err := runCommand("docker", "compose", "-f", composeFile, "up", "-d", "db"); err != nil { // #nosec G204
		return fmt.Errorf("failed to start database: %w", err)
	}
	time.Sleep(2 * time.Second)
	//nolint:forbidigo
	if err := runCommandVerbose("docker", "compose", "-f", composeFile, "up", "-d", "app", "discord"); err != nil { // #nosec G204
		return fmt.Errorf("failed to start services: %w", err)
	}
	PrintSuccess("Services started")

	// 4. Prune old images
	PrintInfo("Cleaning up old images...")
	//nolint:forbidigo
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

	//nolint:forbidigo
	lastTag, err := getCommandOutput("git", "describe", "--tags", "--abbrev=0")
	rangeSpec := "HEAD~5..HEAD"
	title := fmt.Sprintf("Deployment Update (%s)", version)

	if err == nil && lastTag != "" {
		rangeSpec = fmt.Sprintf("%s..HEAD", lastTag)
		title = fmt.Sprintf("Deployment Update (%s -> HEAD)", lastTag)
	}

	//nolint:forbidigo
	notes, _ := getCommandOutput("git", "log", "--pretty=format:• %s (%an)", rangeSpec, "-n", "20") // #nosec G204
	if notes == "" {
		notes = "No new commits in this deployment."
	}

	payload := map[string]interface{}{
		"title":       title,
		"description": notes,
		"color":       65280,
	}

	discordURL := "http://localhost:8082/admin/announce"
	if err := postJSON(discordURL, payload, "", nil); err != nil {
		PrintWarning("Failed to send release notes: %v", err)
		return
	}

	PrintSuccess("Release notes sent successfully")
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
	// Ensure DB is running before backup
	PrintInfo("Ensuring database service is running...")
	//nolint:forbidigo
	if err := runCommand("docker", "compose", "-f", composeFile, "up", "-d", "db"); err != nil {
		return fmt.Errorf("failed to start database service: %w", err)
	}

	// Wait for DB to be ready
	PrintInfo("Waiting for database to be ready...")
	time.Sleep(5 * time.Second)

	// Use docker compose exec which is more robust than finding IDs
	cmd := exec.Command("docker", "compose", "-f", composeFile, "exec", "-T", "db", "pg_dump", "-U", dbUser, "-d", dbName)

	// Pass password if available
	dbPass := os.Getenv("DB_PASSWORD")
	if dbPass != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", dbPass))
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	outfile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer outfile.Close()
	cmd.Stdout = outfile

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pg_dump failed: %w\nStderr: %s", err, stderr.String())
	}
	PrintSuccess("Database backup created: %s", filename)

	// Rotate backups - keep last 5
	if err := RotateBackups("backups", "backup_", 5); err != nil {
		PrintWarning("Failed to rotate backups: %v", err)
	}

	return nil
}

func runSmokeTests(port string) error {
	apiKey := os.Getenv("API_KEY")

	testCases := []struct {
		url          string
		needsAuth    bool
		expectedCode int
	}{
		{url: fmt.Sprintf("http://localhost:%s/healthz", port), needsAuth: false, expectedCode: 200},
		{url: fmt.Sprintf("http://localhost:%s/api/v1/progression/tree", port), needsAuth: true, expectedCode: 200},
	}

	for _, tc := range testCases {
		key := ""
		if tc.needsAuth {
			key = apiKey
		}

		resp, err := makeHTTPRequest("GET", tc.url, nil, key)
		if err != nil {
			PrintWarning("Smoke test failed for %s: %v", tc.url, err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode == tc.expectedCode {
			PrintSuccess("✓ %s responding (%d)", tc.url, resp.StatusCode)
		} else {
			PrintWarning("✗ %s returned %d (expected %d)", tc.url, resp.StatusCode, tc.expectedCode)
		}
	}
	return nil
}

func cleanupOldImages(imageName string) {
	// Validate imageName to prevent command injection
	if imageName == "" {
		PrintWarning("Image name is empty, skipping cleanup")
		return
	}
	if strings.ContainsAny(imageName, ";|&$`\"'") {
		PrintWarning("Invalid characters in image name, skipping cleanup")
		return
	}

	// This is a bit complex to do cross-platform with just exec,
	// simpler to shell out or use docker Go SDK.
	// For now, mirroring the shell script logic roughly.
	//nolint:gosec // G204: imageName is validated above
	cmd := exec.Command("sh", "-c", fmt.Sprintf("docker images \"%s\" --format \"{{.Tag}}\" | grep -v \"latest\" | tail -n +6 | xargs -r -I {} docker rmi \"%s:{}\"", imageName, imageName))
	_ = cmd.Run()
}
