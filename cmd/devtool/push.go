package main

import (
	"fmt"
	"os"
	"strings"
)

type PushCommand struct{}

func (c *PushCommand) Name() string {
	return "push"
}

func (c *PushCommand) Description() string {
	return "Build and push Docker images"
}

func (c *PushCommand) Run(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: devtool push <environment> <version>")
	}

	env := args[0]
	version := args[1]

	if env != envStaging && env != envProduction {
		return fmt.Errorf("environment must be '%s' or '%s'", envStaging, envProduction)
	}

	PrintHeader(fmt.Sprintf("Docker Image Push (%s)", env))
	PrintInfo("Environment: %s", env)
	PrintInfo("Version: %s", version)

	dockerUser := os.Getenv("DOCKER_USER")
	if dockerUser == "" {
		PrintError("DOCKER_USER is not set in .env")
		PrintInfo("Please set DOCKER_USER to your Docker Hub username or registry URL")
		return fmt.Errorf("DOCKER_USER missing")
	}

	imageName := os.Getenv("DOCKER_IMAGE_NAME")
	if imageName == "" {
		imageName = appName
	}
	fullImageName := fmt.Sprintf("%s/%s", dockerUser, imageName)

	PrintInfo("Image: %s", fullImageName)

	// Check Docker Login
	if err := runCommand("docker", "system", "info"); err != nil || !strings.Contains(func() string { out, _ := getCommandOutput("docker", "system", "info"); return out }(), "Username") {
		PrintWarning("Not logged into Docker Hub/Registry. Attempting login...")
		if err := runCommandVerbose("docker", "login"); err != nil {
			return fmt.Errorf("docker login failed: %w", err)
		}
	}

	// Build Image
	PrintInfo("Building image...")
	buildArgs := []string{
		"build",
		"--build-arg", fmt.Sprintf("VERSION=%s", version),
		"-t", fmt.Sprintf("%s:%s", fullImageName, version),
		"-t", fmt.Sprintf("%s:latest-%s", fullImageName, env),
		"-t", fmt.Sprintf("%s:%s", appName, version),
		"-f", "Dockerfile",
		".",
	}
	if err := runCommandVerbose("docker", buildArgs...); err != nil {
		return fmt.Errorf("docker build failed: %w", err)
	}

	// Push Tags
	PrintInfo("Pushing tags to registry...")
	if err := runCommandVerbose("docker", "push", fmt.Sprintf("%s:%s", fullImageName, version)); err != nil {
		return fmt.Errorf("push failed: %w", err)
	}
	if err := runCommandVerbose("docker", "push", fmt.Sprintf("%s:latest-%s", fullImageName, env)); err != nil {
		return fmt.Errorf("push failed: %w", err)
	}

	PrintSuccess("✅ Successfully pushed:")
	PrintSuccess("  - %s:%s", fullImageName, version)
	PrintSuccess("  - %s:latest-%s", fullImageName, env)

	return nil
}
