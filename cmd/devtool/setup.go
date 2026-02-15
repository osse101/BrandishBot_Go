package main

import (
	"fmt"
	"io"
	"os"
)

type SetupCommand struct{}

func (c *SetupCommand) Name() string {
	return "setup"
}

func (c *SetupCommand) Description() string {
	return "Setup development environment"
}

func (c *SetupCommand) Run(args []string) error {
	PrintHeader("Starting Environment Setup")

	// 1. Check Dependencies
	PrintInfo("Step 1/6: Checking dependencies...")
	if err := (&CheckDepsCommand{}).Run(nil); err != nil {
		return err
	}

	// 2. Setup .env
	PrintInfo("Step 2/6: Configuring environment...")
	if _, err := os.Stat(".env"); os.IsNotExist(err) {
		PrintInfo("Creating .env from .env.example...")
		if err := copyFile(".env.example", ".env"); err != nil {
			return fmt.Errorf("failed to create .env: %w", err)
		}
		PrintSuccess(".env created")
		// Reload .env into current process
		// (Main already loaded it, but it didn't exist then. We might need to reload if we want to use the vars immediately)
		// but typically we'd need to restart or use godotenv.Load() again.
		// Since we are in the same process, we can try to rely on defaults or just warn the user.
		PrintInfo("Note: .env created. You might need to re-run if env vars are missing.")
	} else {
		PrintSuccess(".env already exists")
	}

	// 3. Start Docker & DB
	PrintInfo("Step 3/6: Starting database...")
	if err := (&CheckDBCommand{}).Run(nil); err != nil {
		return err
	}

	// 4. Run Migrations
	PrintInfo("Step 4/6: Running migrations...")
	if err := (&MigrateCommand{}).Run([]string{"up"}); err != nil {
		return fmt.Errorf("migrations failed: %w", err)
	}

	// 5. Generate Code
	PrintInfo("Step 5/6: Generating code...")
	//nolint:forbidigo
	if err := runCommandVerbose("make", "generate"); err != nil {
		return fmt.Errorf("code generation failed: %w", err)
	}

	// 6. Install Hooks
	PrintInfo("Step 6/6: Installing git hooks...")
	if err := (&InstallHooksCommand{}).Run(nil); err != nil {
		return fmt.Errorf("failed to install hooks: %w", err)
	}

	PrintSuccess("Setup complete! You can now run 'make run' or 'devtool run'.")
	return nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
