package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func runCheckCoverage(file, thresholdStr string) {
	fmt.Printf("Checking coverage threshold (%s%%)...\n", thresholdStr)

	if _, err := os.Stat(file); os.IsNotExist(err) {
		fmt.Printf("Error: Coverage file '%s' not found.\n", file)
		fmt.Println("Run tests first to generate it.")
		os.Exit(1)
	}

	threshold, err := strconv.ParseFloat(thresholdStr, 64)
	if err != nil {
		fmt.Printf("Error: Invalid threshold '%s'\n", thresholdStr)
		os.Exit(1)
	}

	// Run go tool cover -func=file
	out, err := getCommandOutput("go", "tool", "cover", fmt.Sprintf("-func=%s", file))
	if err != nil {
		fmt.Printf("Error running go tool cover: %v\n", err)
		os.Exit(1)
	}

	// Output format:
	// ...
	// total:  (statements)    82.5%

	lines := strings.Split(out, "\n")
	var totalLine string
	for _, line := range lines {
		if strings.HasPrefix(line, "total:") {
			totalLine = line
			break
		}
	}

	if totalLine == "" {
		fmt.Println("Error: Could not determine coverage from output.")
		os.Exit(1)
	}

	// Parse percentage
	// total:  (statements)    82.5%
	// Split fields
	fields := strings.Fields(totalLine)
	if len(fields) < 3 {
		fmt.Println("Error: Unexpected output format.")
		os.Exit(1)
	}

	pctStr := fields[len(fields)-1] // Last field is percentage
	pctStr = strings.TrimSuffix(pctStr, "%")

	coverage, err := strconv.ParseFloat(pctStr, 64)
	if err != nil {
		fmt.Printf("Error: Could not parse coverage percentage '%s'\n", pctStr)
		os.Exit(1)
	}

	fmt.Printf("Total Coverage: %.1f%%\n", coverage)

	if coverage >= threshold {
		fmt.Println("✅ Coverage meets threshold.")
	} else {
		fmt.Println("❌ Coverage is below threshold.")
		os.Exit(1)
	}
}
