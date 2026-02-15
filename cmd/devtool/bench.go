package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

type BenchCommand struct{}

func (c *BenchCommand) Name() string {
	return "bench"
}

func (c *BenchCommand) Description() string {
	return "Run and manage benchmarks"
}

func (c *BenchCommand) Run(args []string) error {
	if len(args) == 0 {
		return c.runAll()
	}

	subcmd := args[0]
	switch subcmd {
	case "run":
		return c.runAll()
	case "hot":
		return c.runHot()
	case "save":
		return c.save()
	case "baseline":
		return c.baseline()
	case "compare":
		return c.compare()
	case "profile":
		return c.profile()
	default:
		return fmt.Errorf("unknown subcommand: %s", subcmd)
	}
}

func (c *BenchCommand) runAll() error {
	PrintHeader("Running all benchmarks...")
	//nolint:forbidigo
	return runCommandVerbose("go", "test", "-bench=.", "-benchmem", "-benchtime=2s", "./...")
}

func (c *BenchCommand) runHot() error {
	PrintHeader("Running hot path benchmarks...")

	fmt.Println("  → Handler: HandleMessageHandler")
	c.runBenchOrWarn("./internal/handler", "BenchmarkHandler_HandleMessage")

	fmt.Println("  → Service: HandleIncomingMessage")
	c.runBenchOrWarn("./internal/user", "BenchmarkService_HandleIncomingMessage")

	fmt.Println("  → Service: AddItem")
	c.runBenchOrWarn("./internal/user", "BenchmarkService_AddItem")

	fmt.Println("  → Utils: Inventory operations (existing)")
	// This one shouldn't fail silently as it catches all in utils
	//nolint:forbidigo
	return runCommandVerbose("go", "test", "-bench=.", "-benchmem", "-benchtime=2s", "./internal/utils")
}

func (c *BenchCommand) runBenchOrWarn(dir, pattern string) {
	// Validate inputs to prevent command injection
	if dir == "" || pattern == "" {
		fmt.Println("    (invalid benchmark parameters)")
		return
	}
	if strings.ContainsAny(dir, ";|&$`") || strings.ContainsAny(pattern, ";|&$`") {
		fmt.Println("    (invalid characters in benchmark parameters)")
		return
	}

	//nolint:gosec // G204: pattern and dir are validated above
	cmd := exec.Command("go", "test", "-bench="+pattern, "-benchmem", "-benchtime=2s", dir)
	cmd.Stdout = os.Stdout
	// Stderr is discarded to match Makefile's 2>/dev/null
	if err := cmd.Run(); err != nil {
		fmt.Println("    (benchmark not yet implemented)")
	}
}

func (c *BenchCommand) save() error {
	return c.runAndSave(fmt.Sprintf("%s.txt", time.Now().Format("20060102-150405")))
}

func (c *BenchCommand) baseline() error {
	return c.runAndSave("baseline.txt")
}

func (c *BenchCommand) runAndSave(filename string) error {
	PrintHeader("Running benchmarks and saving results...")
	if err := os.MkdirAll("benchmarks/results", 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	path := fmt.Sprintf("benchmarks/results/%s", filename)
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	// We want to write to both stdout and file
	mw := io.MultiWriter(os.Stdout, f)

	cmd := exec.Command("go", "test", "-bench=.", "-benchmem", "-benchtime=2s", "./...")
	cmd.Stdout = mw
	cmd.Stderr = mw // preserve stderr and save to file

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("benchmark execution failed: %w", err)
	}

	PrintSuccess("Results saved to %s", path)
	return nil
}

func (c *BenchCommand) compare() error {
	if _, err := os.Stat("benchmarks/results/baseline.txt"); os.IsNotExist(err) {
		return fmt.Errorf("no baseline found. Run 'devtool bench baseline' first")
	}

	PrintHeader("Running benchmarks and comparing to baseline...")
	if err := os.MkdirAll("benchmarks/results", 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Run current benchmarks
	currentPath := "benchmarks/results/current.txt"

	f, err := os.Create(currentPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	cmd := exec.Command("go", "test", "-bench=.", "-benchmem", "-benchtime=2s", "./...")
	cmd.Stdout = f
	cmd.Stderr = f // redirect stderr too, as per Makefile logic

	_ = cmd.Run() // Ignore error as we want to compare even if some fail
	f.Close()     // Close before reading

	// Check benchstat
	if _, err := exec.LookPath("benchstat"); err == nil {
		cmdStat := exec.Command("benchstat", "benchmarks/results/baseline.txt", currentPath)
		cmdStat.Stdout = os.Stdout
		cmdStat.Stderr = os.Stderr
		return cmdStat.Run()
	}

	PrintWarning("benchstat not installed. Install with: go install golang.org/x/perf/cmd/benchstat@latest")
	fmt.Println("")
	fmt.Println("Showing raw comparison:")
	fmt.Println("======================")
	fmt.Println("BASELINE:")
	c.printHead("benchmarks/results/baseline.txt", 5)
	fmt.Println("")
	fmt.Println("CURRENT:")
	c.printHead("benchmarks/results/current.txt", 5)

	return nil
}

func (c *BenchCommand) printHead(path string, n int) {
	content, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("Error reading %s: %v\n", path, err)
		return
	}
	lines := strings.Split(string(content), "\n")
	count := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "Benchmark") {
			fmt.Println(line)
			count++
			if count >= n {
				break
			}
		}
	}
}

func (c *BenchCommand) profile() error {
	PrintHeader("Profiling hot paths...")
	if err := os.MkdirAll("benchmarks/profiles", 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	fmt.Println("  → CPU profile (if benchmark exists)...")
	// Try Handler
	cmd1 := exec.Command("go", "test", "-bench=BenchmarkHandler_HandleMessage", "-cpuprofile=benchmarks/profiles/cpu.prof", "./internal/handler")
	if err := cmd1.Run(); err != nil {
		// Try Utils
		cmd2 := exec.Command("go", "test", "-bench=BenchmarkAddItems", "-cpuprofile=benchmarks/profiles/cpu.prof", "./internal/utils")
		_ = cmd2.Run()
	}

	fmt.Println("  → Memory profile (if benchmark exists)...")
	// Try Handler
	cmd3 := exec.Command("go", "test", "-bench=BenchmarkHandler_HandleMessage", "-memprofile=benchmarks/profiles/mem.prof", "-benchmem", "./internal/handler")
	if err := cmd3.Run(); err != nil {
		// Try Utils
		cmd4 := exec.Command("go", "test", "-bench=BenchmarkAddItems", "-memprofile=benchmarks/profiles/mem.prof", "-benchmem", "./internal/utils")
		_ = cmd4.Run()
	}

	PrintSuccess("Profiles saved to benchmarks/profiles/")
	fmt.Println("")
	fmt.Println("View CPU profile with:")
	fmt.Println("  go tool pprof -http=:8080 benchmarks/profiles/cpu.prof")
	fmt.Println("View memory profile with:")
	fmt.Println("  go tool pprof -http=:8080 benchmarks/profiles/mem.prof")

	return nil
}
