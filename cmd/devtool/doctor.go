package main

import "fmt"

type DoctorCommand struct{}

func (c *DoctorCommand) Name() string {
	return "doctor"
}

func (c *DoctorCommand) Description() string {
	return "Diagnose environment issues (deps + db)"
}

func (c *DoctorCommand) Run(args []string) error {
	PrintHeader("Running Doctor...")

	hasError := false

	// Run Check Deps
	depsCmd := &CheckDepsCommand{}
	if err := depsCmd.Run(nil); err != nil {
		PrintError("Dependencies check failed: %v", err)
		hasError = true
	} else {
		PrintSuccess("Dependencies OK")
	}

	// Run Check DB
	dbCmd := &CheckDBCommand{}
	if err := dbCmd.Run(nil); err != nil {
		PrintError("Database check failed: %v", err)
		hasError = true
	} else {
		PrintSuccess("Database OK")
	}

	if hasError {
		return fmt.Errorf("doctor found issues")
	}

	PrintSuccess("All systems operational!")
	return nil
}
