package main

import (
	"flag"
	"fmt"
	"os"
)

type GenerateMocksCommand struct{}

func (c *GenerateMocksCommand) Name() string {
	return "mocks"
}

func (c *GenerateMocksCommand) Description() string {
	return "Generate mocks using mockery"
}

func (c *GenerateMocksCommand) Run(args []string) error {
	fs := flag.NewFlagSet("mocks", flag.ContinueOnError)
	clean := fs.Bool("clean", false, "Remove the mocks directory before generating")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *clean {
		PrintInfo("Removing generated mocks...")
		if err := os.RemoveAll("mocks/"); err != nil {
			PrintError("Failed to remove mocks directory: %v", err)
			return err
		}
		PrintSuccess("Removed mocks/ directory")
		return nil
	}

	PrintHeader("Generating mocks...")

	//nolint:forbidigo
	if err := runCommand("go", "run", "github.com/vektra/mockery/v2"); err != nil {
		PrintError("Failed to generate mocks: %v", err)
		return fmt.Errorf("mockery failed: %w", err)
	}

	PrintSuccess("Mocks generated successfully")
	return nil
}
