package main

import (
	"fmt"
	"os"
	"strings"
)

type CheckDepsCommand struct{}

func (c *CheckDepsCommand) Name() string {
	return "check-deps"
}

func (c *CheckDepsCommand) Description() string {
	return "Check for required dependencies"
}

func (c *CheckDepsCommand) Run(args []string) error {
	PrintHeader("Checking dependencies...")

	hasError := false

	// Check Go
	if version, err := getCommandOutput("go", "version"); err == nil {
		parts := strings.Fields(version)
		if len(parts) >= 3 {
			PrintSuccess("Go installed: %s", parts[2])
		} else {
			PrintSuccess("Go installed: %s", version)
		}
	} else {
		PrintError("Go not found!")
		fmt.Println("   Install from: https://go.dev/dl/")
		hasError = true
	}

	// Check Docker
	if version, err := getCommandOutput("docker", "--version"); err == nil {
		parts := strings.Fields(version)
		if len(parts) >= 3 {
			v := strings.TrimRight(parts[2], ",")
			PrintSuccess("Docker installed: %s", v)
		} else {
			PrintSuccess("Docker installed: %s", version)
		}
	} else {
		PrintError("Docker not found!")
		fmt.Println("   Install from: https://docs.docker.com/get-docker/")
		hasError = true
	}

	// Check Docker Compose
	if version, err := getCommandOutput("docker", "compose", "version"); err == nil {
		parts := strings.Fields(version)
		if len(parts) >= 4 {
			PrintSuccess("Docker Compose installed: %s", parts[3])
		} else {
			PrintSuccess("Docker Compose installed: %s", version)
		}
	} else {
		PrintWarning("Docker Compose not found (optional if using 'docker compose')")
	}

	// Check Make
	if version, err := getCommandOutput("make", "--version"); err == nil {
		lines := strings.Split(version, "\n")
		if len(lines) > 0 {
			parts := strings.Fields(lines[0])
			if len(parts) >= 3 {
				PrintSuccess("Make installed: %s", parts[2])
			} else {
				PrintSuccess("Make installed: %s", lines[0])
			}
		}
	} else {
		PrintError("Make not found!")
		fmt.Println("   Install via package manager (e.g., sudo apt install make)")
		hasError = true
	}

	// Check Goose
	if version, err := getCommandOutput("goose", "--version"); err == nil {
		parts := strings.Fields(version)
		if len(parts) >= 1 {
			v := parts[len(parts)-1]
			v = strings.TrimPrefix(v, "version:")
			PrintSuccess("Goose installed: %s", v)
		} else {
			PrintSuccess("Goose installed: %s", version)
		}
	} else {
		// Check GOPATH/bin
		home, _ := os.UserHomeDir()
		goosePath := fmt.Sprintf("%s/go/bin/goose", home)
		if version, err := getCommandOutput(goosePath, "--version"); err == nil {
			parts := strings.Fields(version)
			v := parts[len(parts)-1]
			v = strings.TrimPrefix(v, "version:")
			PrintSuccess("Goose installed (in ~/go/bin): %s", v)
		} else {
			PrintWarning("Goose not found (Recommended for dev)")
			fmt.Println("   Install: go install github.com/pressly/goose/v3/cmd/goose@v3.11.0")
		}
	}

	if hasError {
		return fmt.Errorf("missing dependencies")
	}

	PrintSuccess("Environment check complete!")
	return nil
}
