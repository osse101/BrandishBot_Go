package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if it exists
	_ = godotenv.Load()

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "check-deps":
		runCheckDeps()
	case "check-db":
		runCheckDB()
	case "check-coverage":
		if len(os.Args) < 4 {
			fmt.Println("Usage: check-coverage <coverage_file> <threshold>")
			os.Exit(1)
		}
		runCheckCoverage(os.Args[2], os.Args[3])
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: devtool <command> [args...]")
	fmt.Println("Commands:")
	fmt.Println("  check-deps      Check for required dependencies")
	fmt.Println("  check-db        Check if database is running and ready")
	fmt.Println("  check-coverage  Check test coverage against a threshold")
}
