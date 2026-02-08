package main

import (
	"fmt"
	"time"
)

type HealthCheckCommand struct{}

func (c *HealthCheckCommand) Name() string {
	return "health-check"
}

func (c *HealthCheckCommand) Description() string {
	return "Check application health"
}

func (c *HealthCheckCommand) Run(args []string) error {
	env := "production"
	if len(args) > 0 {
		env = args[0]
	}

	if env != "staging" && env != "production" {
		// Try to see if env is actually just "health-check" called without args
		// But args passed to Run() are skipping the command name itself.
		// So if args is empty, default to production.
	}

	PrintHeader(fmt.Sprintf("Health Check (%s)", env))

	if err := checkHealth(env); err != nil {
		PrintError("Health check failed: %v", err)
		return err
	}

	// Also check response time
	start := time.Now()
	if err := checkHealth(env); err != nil {
		return err
	}
	duration := time.Since(start)

	if duration > 1*time.Second {
		PrintWarning("Health check warning: slow response time (%v)", duration)
	} else {
		PrintSuccess("Health check passed (response time: %v)", duration)
	}

	return nil
}
