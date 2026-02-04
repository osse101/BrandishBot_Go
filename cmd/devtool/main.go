package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if it exists
	_ = godotenv.Load()

	registry := NewRegistry()
	registry.Register(&CheckDepsCommand{})
	registry.Register(&CheckDBCommand{})
	registry.Register(&CheckCoverageCommand{})
	registry.Register(&TestSecurityCommand{})
	registry.Register(&TestMigrationsCommand{})
	registry.Register(&DoctorCommand{})
	registry.Register(&SetupCommand{})

	if len(os.Args) < 2 {
		registry.PrintHelp()
		os.Exit(1)
	}

	commandName := os.Args[1]
	if commandName == "help" {
		registry.PrintHelp()
		return
	}

	cmd, ok := registry.Get(commandName)
	if !ok {
		fmt.Printf("Unknown command: %s\n", commandName)
		registry.PrintHelp()
		os.Exit(1)
	}

	args := os.Args[2:]
	if err := cmd.Run(args); err != nil {
		// Ensure error is printed if the command failed
		// Some commands print their own errors using UI helpers, but we print here to be safe
		// and ensure the user knows why it exited with 1.
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
