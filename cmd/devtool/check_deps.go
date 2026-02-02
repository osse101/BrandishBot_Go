package main

import (
	"fmt"
	"os"
	"strings"
)

func runCheckDeps() {
	fmt.Println("Checking dependencies...")

	hasError := false

	// Check Go
	if version, err := getCommandOutput("go", "version"); err == nil {
		// Output: go version go1.21.0 linux/amd64
		parts := strings.Fields(version)
		if len(parts) >= 3 {
			fmt.Printf("✅ Go installed: %s\n", parts[2])
		} else {
			fmt.Printf("✅ Go installed: %s\n", version)
		}
	} else {
		fmt.Println("❌ Go not found!")
		fmt.Println("   Install from: https://go.dev/dl/")
		hasError = true
	}

	// Check Docker
	if version, err := getCommandOutput("docker", "--version"); err == nil {
		// Output: Docker version 24.0.5, build ced0996
		parts := strings.Fields(version)
		if len(parts) >= 3 {
			v := strings.TrimRight(parts[2], ",")
			fmt.Printf("✅ Docker installed: %s\n", v)
		} else {
			fmt.Printf("✅ Docker installed: %s\n", version)
		}
	} else {
		fmt.Println("❌ Docker not found!")
		fmt.Println("   Install from: https://docs.docker.com/get-docker/")
		hasError = true
	}

	// Check Docker Compose
	// Try 'docker compose' first
	if version, err := getCommandOutput("docker", "compose", "version"); err == nil {
		// Output: Docker Compose version v2.20.2
		parts := strings.Fields(version)
		if len(parts) >= 4 {
			fmt.Printf("✅ Docker Compose installed: %s\n", parts[3])
		} else {
			fmt.Printf("✅ Docker Compose installed: %s\n", version)
		}
	} else {
		fmt.Println("⚠️  Docker Compose not found (optional if using 'docker compose')")
	}

	// Check Make
	if version, err := getCommandOutput("make", "--version"); err == nil {
		// Output: GNU Make 4.3 ...
		lines := strings.Split(version, "\n")
		if len(lines) > 0 {
			parts := strings.Fields(lines[0])
			if len(parts) >= 3 {
				fmt.Printf("✅ Make installed: %s\n", parts[2])
			} else {
				fmt.Printf("✅ Make installed: %s\n", lines[0])
			}
		}
	} else {
		fmt.Println("❌ Make not found!")
		fmt.Println("   Install via package manager (e.g., sudo apt install make)")
		hasError = true
	}

	// Check Goose
	if version, err := getCommandOutput("goose", "--version"); err == nil {
		// Output: goose version:v3.15.0
		parts := strings.Fields(version)
		if len(parts) >= 1 {
			// format might be "goose version:v3.15.0" or "goose version v3.15.0"
			v := parts[len(parts)-1]
			v = strings.TrimPrefix(v, "version:")
			fmt.Printf("✅ Goose installed: %s\n", v)
		} else {
			fmt.Printf("✅ Goose installed: %s\n", version)
		}
	} else {
		// Check GOPATH/bin
		home, _ := os.UserHomeDir()
		goosePath := fmt.Sprintf("%s/go/bin/goose", home)
		if version, err := getCommandOutput(goosePath, "--version"); err == nil {
			parts := strings.Fields(version)
			v := parts[len(parts)-1]
			v = strings.TrimPrefix(v, "version:")
			fmt.Printf("✅ Goose installed (in ~/go/bin): %s\n", v)
		} else {
			fmt.Println("⚠️  Goose not found (Recommended for dev)")
			fmt.Println("   Install: go install github.com/pressly/goose/v3/cmd/goose@v3.11.0")
		}
	}

	if hasError {
		os.Exit(1)
	}

	fmt.Println("\n🎉 Environment check complete!")
}
