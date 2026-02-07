package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

type PreCommitCommand struct{}

func (c *PreCommitCommand) Name() string {
	return "pre-commit"
}

func (c *PreCommitCommand) Description() string {
	return "Run pre-commit checks (secrets, fmt, generate, lint, test)"
}

func (c *PreCommitCommand) Run(args []string) error {
	PrintHeader("Running pre-commit checks...")

	// 1. Get staged files
	stagedFiles, err := getStagedFiles()
	if err != nil {
		return fmt.Errorf("failed to get staged files: %w", err)
	}

	if len(stagedFiles) == 0 {
		PrintInfo("No staged files found.")
		return nil
	}

	// 2. Secret Scanning
	if err := checkSecrets(stagedFiles); err != nil {
		return err
	}

	// 3. Go Format
	if err := runGoFmt(stagedFiles); err != nil {
		return err
	}

	// 4. Generate Check
	if err := checkGenerate(stagedFiles); err != nil {
		return err
	}

	// 5. Linting
	if err := runLinter(); err != nil {
		return err
	}

	// 6. Unit Tests
	if err := runUnitTests(); err != nil {
		return err
	}

	PrintSuccess("All pre-commit checks passed!")
	return nil
}

func getStagedFiles() ([]string, error) {
	out, err := getCommandOutput("git", "diff", "--cached", "--name-only", "--diff-filter=ACM")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return []string{}, nil
	}
	return strings.Split(out, "\n"), nil
}

func checkSecrets(files []string) error {
	PrintInfo("Checking for secrets...")

	// Regex from the bash script
	// ((mfa\.[a-z0-9_-]{20,})|([a-z0-9_-]{24}\.[a-z0-9_-]{6}\.[a-z0-9_-]{27}))|(\b(password|secret|api_key|token|client_id|client_secret)\b\s*[:=]\s*['"][^'"]+['"])
	pattern := `((mfa\.[a-z0-9_-]{20,})|([a-z0-9_-]{24}\.[a-z0-9_-]{6}\.[a-z0-9_-]{27}))|(\b(password|secret|api_key|token|client_id|client_secret)\b\s*[:=]\s*['"][^'"]+['"])`
	re := regexp.MustCompile(pattern)

	found := false
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			// If file doesn't exist (e.g. deleted but staged as modified? unlikely with diff-filter=ACM), skip
			continue
		}
		if re.Match(content) {
			PrintError("Potential secret found in %s", file)
			found = true
		}
	}

	if found {
		return fmt.Errorf("secrets found in staged files")
	}
	return nil
}

func runGoFmt(files []string) error {
	var goFiles []string
	for _, f := range files {
		if strings.HasSuffix(f, ".go") {
			goFiles = append(goFiles, f)
		}
	}

	if len(goFiles) == 0 {
		return nil
	}

	PrintInfo("Running go fmt...")
	for _, f := range goFiles {
		if err := runCommand("go", "fmt", f); err != nil {
			return fmt.Errorf("go fmt failed for %s: %w", f, err)
		}
		if err := runCommand("git", "add", f); err != nil {
			return fmt.Errorf("git add failed for %s: %w", f, err)
		}
	}
	return nil
}

func checkGenerate(files []string) error {
	// Trigger files: .sql$, interfaces.go$, go.mod$, go.sum$, progression_tree.json$
	shouldRun := false
	triggerPattern := regexp.MustCompile(`(\.sql$|interfaces\.go$|go\.mod$|go\.sum$|progression_tree\.json$)`)

	for _, f := range files {
		if triggerPattern.MatchString(f) {
			shouldRun = true
			break
		}
	}

	if !shouldRun {
		return nil
	}

	PrintInfo("Running 'make generate'...")
	if err := runCommand("make", "generate"); err != nil {
		return fmt.Errorf("make generate failed: %w", err)
	}

	// Check for unstaged changes
	if err := runCommand("git", "diff", "--exit-code"); err != nil {
		// git diff --exit-code returns 1 if there are differences
		PrintError("'make generate' produced changes that are not staged.")
		PrintWarning("Please stage the updated files (mocks, sqlc, go.mod) and try again.")
		return fmt.Errorf("generated files are not staged")
	}

	return nil
}

func runLinter() error {
	PrintInfo("Running linter on changes...")
	// Use go run to ensure we use the version pinned in tools.go
	cmd := exec.Command("go", "run", "github.com/golangci/golangci-lint/cmd/golangci-lint", "run", "--new-from-rev=HEAD", "./...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("linter failed")
	}
	return nil
}

func runUnitTests() error {
	PrintInfo("Running unit tests...")
	if err := runCommandVerbose("make", "unit"); err != nil {
		return fmt.Errorf("unit tests failed")
	}
	return nil
}
