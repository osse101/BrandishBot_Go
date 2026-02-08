package main

import (
	"fmt"
	"strings"
)

type CheckDepsCommand struct{}

func (c *CheckDepsCommand) Name() string {
	return "check-deps"
}

func (c *CheckDepsCommand) Description() string {
	return "Check for required dependencies"
}

type toolCheck struct {
	Name    string
	Package string
	CmdArg  string
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

	// Check Go Tools (from tools.go)
	PrintHeader("Checking Go Tools (via go run)...")
	tools := []toolCheck{
		{Name: "Goose", Package: "github.com/pressly/goose/v3/cmd/goose", CmdArg: "--version"},
		{Name: "SQLC", Package: "github.com/sqlc-dev/sqlc/cmd/sqlc", CmdArg: "version"},
		{Name: "Swag", Package: "github.com/swaggo/swag/cmd/swag", CmdArg: "--version"},
		{Name: "Mockery", Package: "github.com/vektra/mockery/v2", CmdArg: "--version"},
		{Name: "GolangCI-Lint", Package: "github.com/golangci/golangci-lint/cmd/golangci-lint", CmdArg: "--version"},
		{Name: "Benchstat", Package: "golang.org/x/perf/cmd/benchstat", CmdArg: "-h"},
	}

	for _, t := range tools {
		// Use go run to check if the tool is runnable/installable
		// We ignore the output content mostly, just care about exit code 0
		cmd := []string{"run", t.Package, t.CmdArg}
		if err := runCommand("go", cmd...); err == nil {
			PrintSuccess("%s ready", t.Name)
		} else {
			PrintError("%s check failed", t.Name)
			fmt.Printf("   Failed to run: go run %s %s\n", t.Package, t.CmdArg)
			hasError = true
		}
	}

	if hasError {
		return fmt.Errorf("missing dependencies")
	}

	PrintSuccess("Environment check complete!")
	return nil
}
