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
	registry.Register(&MigrateCommand{})
	registry.Register(&DoctorCommand{})
	registry.Register(&SetupCommand{})
	registry.Register(&BenchCommand{})
	registry.Register(&PreCommitCommand{})
	registry.Register(&InstallHooksCommand{})
	registry.Register(&DeployCommand{})
	registry.Register(&RollbackCommand{})
	registry.Register(&HealthCheckCommand{})
	registry.Register(&PushCommand{})
	registry.Register(&TestSSECommand{})
	registry.Register(&BuildCommand{})
	registry.Register(&WaitForDBCommand{})
	registry.Register(&EntrypointCommand{})
	registry.Register(&SeedCommand{})
	registry.Register(&CheckCommentsCommand{})
	registry.Register(&DebugDBSessionsCommand{})
	registry.Register(&AnalyzeLogsCommand{})
	registry.Register(&ScenarioCommand{})
	registry.Register(&TestLogsCommand{})
	registry.Register(&TestLootboxCommand{})
	registry.Register(&GenerateMocksCommand{})
	registry.Register(&TestCommand{})

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
		// Fallback error printing if the command didn't handle it.
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
