package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type CheckCoverageCommand struct{}

func (c *CheckCoverageCommand) Name() string {
	return "check-coverage"
}

func (c *CheckCoverageCommand) Description() string {
	return "Check test coverage against a threshold"
}

func (c *CheckCoverageCommand) Run(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: check-coverage <coverage_file> <threshold>")
	}

	file := args[0]
	thresholdStr := args[1]

	PrintHeader(fmt.Sprintf("Checking coverage threshold (%s%%)...", thresholdStr))

	if _, err := os.Stat(file); os.IsNotExist(err) {
		PrintError("Coverage file '%s' not found.", file)
		PrintInfo("Run tests first to generate it.")
		return fmt.Errorf("coverage file not found")
	}

	threshold, err := strconv.ParseFloat(thresholdStr, 64)
	if err != nil {
		return fmt.Errorf("invalid threshold '%s'", thresholdStr)
	}

	// Run go tool cover -func=file
	out, err := getCommandOutput("go", "tool", "cover", fmt.Sprintf("-func=%s", file))
	if err != nil {
		return fmt.Errorf("error running go tool cover: %v", err)
	}

	lines := strings.Split(out, "\n")
	var totalLine string
	for _, line := range lines {
		if strings.HasPrefix(line, "total:") {
			totalLine = line
			break
		}
	}

	if totalLine == "" {
		return fmt.Errorf("could not determine coverage from output")
	}

	fields := strings.Fields(totalLine)
	if len(fields) < 3 {
		return fmt.Errorf("unexpected output format")
	}

	pctStr := fields[len(fields)-1] // Last field is percentage
	pctStr = strings.TrimSuffix(pctStr, "%")

	coverage, err := strconv.ParseFloat(pctStr, 64)
	if err != nil {
		return fmt.Errorf("could not parse coverage percentage '%s'", pctStr)
	}

	PrintInfo("Total Coverage: %.1f%%", coverage)

	if coverage >= threshold {
		PrintSuccess("Coverage meets threshold.")
		return nil
	} else {
		PrintError("Coverage is below threshold.")
		return fmt.Errorf("coverage below threshold")
	}
}
