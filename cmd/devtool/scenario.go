package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/osse101/BrandishBot_Go/internal/handler"
	"github.com/osse101/BrandishBot_Go/internal/scenario"
)

type ScenarioCommand struct{}

func (c *ScenarioCommand) Name() string {
	return "scenario"
}

func (c *ScenarioCommand) Description() string {
	return "Manage and run simulation scenarios"
}

func (c *ScenarioCommand) Run(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("subcommand required: list, run")
	}

	subcmd := args[0]
	switch subcmd {
	case "list":
		return c.runList()
	case "run":
		if len(args) < 2 {
			return fmt.Errorf("scenario ID required")
		}
		return c.runRun(args[1], args[2:])
	default:
		return fmt.Errorf("unknown subcommand: %s", subcmd)
	}
}

func (c *ScenarioCommand) runList() error {
	PrintHeader("Available Scenarios")

	apiURL := getAPIURL()
	apiKey := getAPIKey()

	var data handler.ScenariosResponse
	if err := getJSON(fmt.Sprintf("%s/api/v1/admin/simulate/scenarios", apiURL), apiKey, &data); err != nil {
		return fmt.Errorf("failed to fetch scenarios: %w", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tName\tFeature\tSteps")
	fmt.Fprintln(w, strings.Repeat("-", 80))

	for _, s := range data.Scenarios {
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\n", s.ID, s.Name, s.Feature, s.StepCount)
	}
	w.Flush()

	return nil
}

func (c *ScenarioCommand) runRun(scenarioID string, extraArgs []string) error {
	PrintHeader(fmt.Sprintf("Running Scenario: %s", scenarioID))

	apiURL := getAPIURL()
	apiKey := getAPIKey()

	params := c.parseParams(extraArgs)

	reqBody := handler.RunScenarioRequest{
		ScenarioID: scenarioID,
		Parameters: params,
	}

	var result scenario.ExecutionResult
	if err := postJSON(fmt.Sprintf("%s/api/v1/admin/simulate/run", apiURL), reqBody, apiKey, &result); err != nil {
		return fmt.Errorf("failed to execute scenario: %w", err)
	}

	c.printExecutionResult(result)

	if !result.Success {
		return fmt.Errorf("scenario failed")
	}

	return nil
}

func (c *ScenarioCommand) parseParams(extraArgs []string) map[string]interface{} {
	params := make(map[string]interface{})
	for _, arg := range extraArgs {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) == 2 {
			params[parts[0]] = parts[1]
		}
	}
	return params
}

func (c *ScenarioCommand) printExecutionResult(result scenario.ExecutionResult) {
	fmt.Printf("Scenario: %s\n", result.ScenarioName)
	fmt.Printf("Duration: %d ms\n", result.DurationMS)

	if result.Success {
		PrintSuccess("Status: SUCCESS")
	} else {
		PrintError("Status: FAILED")
		if result.Error != "" {
			PrintError("Error: %s", result.Error)
		}
	}

	fmt.Println("\nSteps:")
	for _, step := range result.Steps {
		status := "✓"
		if !step.Success {
			status = "✗"
		}
		fmt.Printf("  %s %s (%d ms)\n", status, step.StepName, step.DurationMS)

		if !step.Success && step.Error != "" {
			fmt.Printf("    Error: %s\n", step.Error)
		}

		if len(step.Assertions) > 0 {
			for _, assert := range step.Assertions {
				assertStatus := "✓"
				if !assert.Passed {
					assertStatus = "✗"
				}
				fmt.Printf("    %s Assertion (%s): %s\n", assertStatus, assert.Type, assert.Path)
				if !assert.Passed {
					fmt.Printf("      Reason: %s\n", assert.Reason)
					if assert.Error != "" {
						fmt.Printf("      Error: %s\n", assert.Error)
					}
				}
			}
		}
	}
}

