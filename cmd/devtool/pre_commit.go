package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
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

	// 1. Migration Protections
	// Run this before checking stagedFiles count because deletions/renames
	// might not be in the ACM filter but still need protection.
	if err := checkMigrationProtections(); err != nil {
		return err
	}

	// 2. Get staged files (Added, Copied, Modified)
	stagedFiles, err := getStagedFiles()
	if err != nil {
		return fmt.Errorf("failed to get staged files: %w", err)
	}

	if len(stagedFiles) == 0 {
		PrintInfo("No other staged changes found.")
		return nil
	}

	// 3. Secret Scanning
	if err := checkSecrets(stagedFiles); err != nil {
		return err
	}

	// 4. Go Format
	if err := runGoFmt(stagedFiles); err != nil {
		return err
	}

	// 5. Generate Check
	if err := checkGenerate(stagedFiles); err != nil {
		return err
	}

	// 6. Linting
	if err := runLinter(); err != nil {
		return err
	}

	// 7. Unit Tests
	if err := runUnitTests(); err != nil {
		return err
	}

	// 8. Large File Sentinel
	if err := checkLargeFiles(stagedFiles); err != nil {
		return err
	}

	// 9. Env Template Sync
	if err := checkEnvSync(); err != nil {
		return err
	}

	PrintSuccess("All pre-commit checks passed!")
	return nil
}

func getStagedFiles() ([]string, error) {
	//nolint:forbidigo
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
		//nolint:forbidigo
		if err := runCommand("go", "fmt", f); err != nil { // #nosec G204
			return fmt.Errorf("go fmt failed for %s: %w", f, err)
		}
		//nolint:forbidigo
		if err := runCommand("git", "add", f); err != nil { // #nosec G204
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
	//nolint:forbidigo
	if err := runCommand("make", "generate"); err != nil {
		return fmt.Errorf("make generate failed: %w", err)
	}

	// Check for unstaged changes
	//nolint:forbidigo
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
	PrintInfo("Analyzing changed packages for unit tests...")

	packages, err := getChangedPackages(true) // true = check staged changes
	if err != nil {
		PrintWarning("Failed to detect changed packages: %v. Running all tests.", err)
		packages = []string{"./..."}
	}

	if len(packages) == 0 {
		PrintInfo("No Go packages changed. Skipping unit tests.")
		return nil
	}

	PrintInfo("Running unit tests on: %v", packages)

	args := []string{"test", "-short"}
	args = append(args, packages...)

	//nolint:forbidigo
	if err := runCommandVerbose("go", args...); err != nil {
		return fmt.Errorf("unit tests failed")
	}
	return nil
}

func checkMigrationProtections() error {
	PrintInfo("Checking migration protections...")

	_, newFiles, err := detectMigrationSquashAndChanges()
	if err != nil {
		return err
	}

	if len(newFiles) == 0 {
		return nil
	}

	if err := checkDestructiveSQL(newFiles); err != nil {
		return err
	}

	return verifyMigrationSequence(newFiles)
}

func detectMigrationSquashAndChanges() (bool, []string, error) {
	//nolint:forbidigo
	out, err := getCommandOutput("git", "diff", "--cached", "--name-status")
	if err != nil {
		return false, nil, fmt.Errorf("failed to get git status: %w", err)
	}

	lines := strings.Split(out, "\n")
	isSquash := os.Getenv("ALLOW_MIGRATION_SQUASH") == "1" || hasArchiveChanges(lines)

	var newFiles []string
	for _, line := range lines {
		if line == "" {
			continue
		}
		path, status, skip := parseGitStatusLine(line)
		if skip {
			continue
		}

		if strings.HasPrefix(status, "A") {
			newFiles = append(newFiles, path)
			continue
		}

		if isSquash && (strings.HasPrefix(status, "D") || strings.HasPrefix(status, "R")) {
			continue
		}

		return false, nil, reportMigrationError(status, path, isSquash)
	}

	return isSquash, newFiles, nil
}

func hasArchiveChanges(lines []string) bool {
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		// The last column is the path (or destination path for R)
		if len(parts) >= 2 && strings.HasPrefix(parts[len(parts)-1], "migrations/archive/") {
			return true
		}
	}
	return false
}

func parseGitStatusLine(line string) (path string, status string, skip bool) {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return "", "", true
	}
	status = parts[0]
	path = parts[1]

	if !strings.HasPrefix(path, "migrations/") || strings.HasPrefix(path, "migrations/archive/") || !strings.HasSuffix(path, ".sql") {
		return "", "", true
	}
	return path, status, false
}

func reportMigrationError(status, path string, isSquash bool) error {
	action := "edited"
	if strings.HasPrefix(status, "D") {
		action = "deleted"
	} else if strings.HasPrefix(status, "R") {
		action = "renamed"
	}

	PrintError("Migration files may only be added, not %s: %s (status: %s)", action, path, status)
	if !isSquash {
		PrintInfo("If you are purposefully squashing migrations, move the old files to 'migrations/archive/' or set ALLOW_MIGRATION_SQUASH=1")
	}
	return fmt.Errorf("migration file protection: restricted operation")
}

func verifyMigrationSequence(newMigrationFiles []string) error {
	// 2. Check sequence for new migrations
	files, err := os.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("failed to read migrations: %w", err)
	}

	var allPrefixes []int
	re := regexp.MustCompile(`^(\d{4})_`)

	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".sql") {
			continue
		}
		match := re.FindStringSubmatch(f.Name())
		if len(match) == 2 {
			num, _ := strconv.Atoi(match[1])
			allPrefixes = append(allPrefixes, num)
		}
	}

	if len(allPrefixes) == 0 {
		return nil
	}

	sort.Ints(allPrefixes)

	// Identify which prefixes belong to the newly added files
	newPrefixes := make(map[int]bool)
	for _, path := range newMigrationFiles {
		filename := path[strings.LastIndex(path, "/")+1:]
		match := re.FindStringSubmatch(filename)
		if len(match) == 2 {
			num, _ := strconv.Atoi(match[1])
			newPrefixes[num] = true
		} else {
			PrintError("New migration file name must start with 4 digits: %s", filename)
			return fmt.Errorf("invalid migration filename")
		}
	}

	// Find the max prefix that is NOT one of the new migrations
	maxExisting := 0
	for _, p := range allPrefixes {
		if !newPrefixes[p] {
			if p > maxExisting {
				maxExisting = p
			}
		}
	}

	// The new prefixes must be sequential and follow maxExisting
	sortNewPrefixes := make([]int, 0, len(newPrefixes))
	for p := range newPrefixes {
		sortNewPrefixes = append(sortNewPrefixes, p)
	}
	sort.Ints(sortNewPrefixes)

	for i, p := range sortNewPrefixes {
		expected := maxExisting + i + 1
		if p != expected {
			PrintError("New migration %04d is out of sequence. Expected %04d based on current max %04d.", p, expected, maxExisting)
			return fmt.Errorf("migration sequence gap or duplicate")
		}
	}

	PrintSuccess("Migration sequence verified (%d new migrations).", len(sortNewPrefixes))
	return nil
}

func checkDestructiveSQL(newFiles []string) error {
	PrintInfo("Checking for destructive SQL...")

	destructiveRegex := regexp.MustCompile(`(?i)\b(DROP\s+TABLE|DROP\s+COLUMN|TRUNCATE|DROP\s+DATABASE)\b`)
	ignoreComment := "-- skip-destructive-check"

	for _, file := range newFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		contentStr := string(content)
		// Only check the "Up" part of the migration - Down sections naturally contain DROP commands
		checkContent := contentStr
		if idx := strings.Index(contentStr, "-- +goose Down"); idx != -1 {
			checkContent = contentStr[:idx]
		}

		if destructiveRegex.MatchString(checkContent) && !strings.Contains(contentStr, ignoreComment) {
			PrintError("Destructive SQL command found in the 'Up' section of %s", file)
			PrintInfo("Only allow deletions if absolutely necessary. To bypass this check, add comment: %s", ignoreComment)
			return fmt.Errorf("destructive SQL found")
		}
	}

	return nil
}

func checkLargeFiles(files []string) error {
	PrintInfo("Checking file sizes...")
	const maxSize = 2 * 1024 * 1024 // 2MB

	for _, file := range files {
		if strings.HasPrefix(file, "media/") {
			continue
		}

		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		if !info.IsDir() && info.Size() > maxSize {
			PrintError("File too large: %s (%.2f MB)", file, float64(info.Size())/(1024*1024))
			PrintInfo("Large files (over 2MB) are blocked from being committed to the repo.")
			return fmt.Errorf("file size limit exceeded")
		}
	}
	return nil
}

func checkEnvSync() error {
	PrintInfo("Checking env template sync...")

	// 1. Get keys from .env.example
	exampleContent, err := os.ReadFile(".env.example")
	if err != nil {
		return fmt.Errorf("failed to read .env.example: %w", err)
	}

	exampleKeys := make(map[string]bool)
	lines := strings.Split(string(exampleContent), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if parts := strings.SplitN(line, "=", 2); len(parts) >= 1 {
			key := strings.TrimSpace(parts[0])
			if key != "" {
				exampleKeys[key] = true
			}
		}
	}

	// 2. Find keys in codebase
	envRegex := regexp.MustCompile(`os\.Getenv\("([^"]+)"\)`)
	foundMissing := false

	// Use git grep to find os.Getenv calls in .go files (much faster)
	//nolint:forbidigo
	out, _ := getCommandOutput("git", "grep", "-E", `os\.Getenv\("([^"]+)"\)`, "--", "*.go")

	// Environment variables that are optional and don't need to be in .env.example
	// These are typically runtime overrides or container-specific variables
	optionalEnvVars := map[string]bool{
		"DB_URL":                 true, // Optional database connection string (overrides individual DB_* vars)
		"ALLOW_MIGRATION_SQUASH": true, // Development-only flag for migration squashing
		"CREATE_BACKUP":          true, // Optional backup creation flag
		"TEST_DB_CONN":           true, // Optional test database connection string
	}

	lines = strings.Split(out, "\n")
	for _, line := range lines {
		matches := envRegex.FindAllStringSubmatch(line, -1)
		for _, m := range matches {
			if len(m) == 2 {
				key := m[1]
				// Skip optional environment variables
				if optionalEnvVars[key] {
					continue
				}
				if !exampleKeys[key] {
					PrintError("Environment variable '%s' used in code but missing from .env.example", key)
					foundMissing = true
				}
			}
		}
	}

	if foundMissing {
		return fmt.Errorf("env template out of sync")
	}

	PrintSuccess(".env.example is in sync with codebase.")
	return nil
}
