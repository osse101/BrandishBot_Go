package main

import (
	"fmt"
	"os"
	"time"
)

type BuildCommand struct{}

func (c *BuildCommand) Name() string {
	return "build"
}

func (c *BuildCommand) Description() string {
	return "Builds the application binaries to bin/ directory"
}

func (c *BuildCommand) Run(args []string) error {
	PrintHeader("Building Binaries")

	// Create bin directory
	if err := os.MkdirAll("bin", 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	// Gather version info
	//nolint:forbidigo
	version, _ := getCommandOutput("git", "describe", "--tags", "--always", "--dirty")
	if version == "" {
		version = "dev"
	}

	buildTime := time.Now().UTC().Format("2006-01-02_15:04")

	//nolint:forbidigo
	gitCommit, _ := getCommandOutput("git", "rev-parse", "--short", "HEAD")
	if gitCommit == "" {
		gitCommit = "unknown"
	}

	ldflags := fmt.Sprintf(
		"-X github.com/osse101/BrandishBot_Go/internal/handler.Version=%s "+
			"-X github.com/osse101/BrandishBot_Go/internal/handler.BuildTime=%s "+
			"-X github.com/osse101/BrandishBot_Go/internal/handler.GitCommit=%s",
		version, buildTime, gitCommit,
	)

	// Build App
	PrintInfo("Building bin/app...")
	//nolint:forbidigo
	if err := runCommand("go", "build", "-ldflags", ldflags, "-o", "bin/app", "./cmd/app"); err != nil {
		return fmt.Errorf("failed to build app: %w", err)
	}
	PrintSuccess("Built: bin/app")

	// Build Discord Bot
	PrintInfo("Building bin/discord_bot...")
	//nolint:forbidigo
	if err := runCommand("go", "build", "-ldflags", ldflags, "-o", "bin/discord_bot", "./cmd/discord"); err != nil {
		return fmt.Errorf("failed to build discord_bot: %w", err)
	}
	PrintSuccess("Built: bin/discord_bot")

	return nil
}
