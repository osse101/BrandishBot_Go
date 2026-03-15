package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
)

func getCoveragePercent(file string) (float64, error) {
	// Run go tool cover -func=file
	//nolint:forbidigo // file is validated in parseConfig
	out, err := getCommandOutput("go", "tool", "cover", fmt.Sprintf("-func=%s", file)) // #nosec G204
	if err != nil {
		return 0, fmt.Errorf("error running go tool cover: %w", err)
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
		return 0, fmt.Errorf("could not determine coverage from output")
	}

	fields := strings.Fields(totalLine)
	if len(fields) < 3 {
		return 0, fmt.Errorf("unexpected output format")
	}

	pctStr := fields[len(fields)-1] // Last field is percentage
	pctStr = strings.TrimSuffix(pctStr, "%")

	coverage, err := strconv.ParseFloat(pctStr, 64)
	if err != nil {
		return 0, fmt.Errorf("could not parse coverage percentage '%s'", pctStr)
	}

	return coverage, nil
}

func generateHTMLReport(file string) error {
	htmlFile := filepath.Clean(strings.TrimSuffix(file, ".out") + ".html")

	// Extra validation for htmlFile
	if strings.Contains(htmlFile, "..") || strings.HasPrefix(htmlFile, "/") {
		return fmt.Errorf("invalid HTML report path '%s'", htmlFile)
	}

	PrintInfo("Generating HTML report: %s", htmlFile)
	// #nosec G204 - file and htmlFile are validated
	cmd := exec.Command("go", "tool", "cover", "-html="+file, "-o", htmlFile)
	if err := cmd.Run(); err != nil {
		return err
	}
	PrintSuccess("HTML report generated: %s", htmlFile)
	return nil
}

type pkgCoverage struct {
	Name     string
	Coverage float64
	Time     string
}

func printPackageCoverageTable(output string) {
	re := regexp.MustCompile(`ok\s+([^\s]+)\s+([0-9.]+)s\s+coverage:\s+(.+)`)
	lines := strings.Split(output, "\n")
	var stats []pkgCoverage

	for _, line := range lines {
		matches := re.FindStringSubmatch(line)
		if len(matches) == 4 {
			pkg := matches[1]
			timeStr := matches[2]
			covStr := matches[3]

			// Handle "coverage: [no statements]"
			if strings.Contains(covStr, "[no statements]") {
				continue
			}

			// "coverage: 50.0% of statements"
			parts := strings.Fields(covStr)
			if len(parts) > 0 {
				valStr := strings.TrimSuffix(parts[0], "%")
				val, err := strconv.ParseFloat(valStr, 64)
				if err == nil {
					stats = append(stats, pkgCoverage{
						Name:     pkg,
						Coverage: val,
						Time:     timeStr + "s",
					})
				}
			}
		}
	}

	if len(stats) == 0 {
		return
	}

	// Sort by coverage (ascending)
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Coverage < stats[j].Coverage
	})

	fmt.Println("\nPackage Coverage Summary:")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Package\tCoverage\tTime")
	fmt.Fprintln(w, strings.Repeat("-", 60))

	for _, s := range stats {
		color := "\033[31m" // Red
		if s.Coverage >= 80 {
			color = "\033[32m" // Green
		} else if s.Coverage >= 50 {
			color = "\033[33m" // Yellow
		}
		reset := "\033[0m"

		// Truncate package name if too long, keeping the end
		name := s.Name
		if len(name) > 50 {
			name = "..." + name[len(name)-47:]
		}

		fmt.Fprintf(w, "%s\t%s%.1f%%%s\t%s\n", name, color, s.Coverage, reset, s.Time)
	}
	w.Flush()
	fmt.Println()
}

type funcCoverage struct {
	Location string
	Name     string
	Coverage float64
}

func printTopMissingFunctions(file string) error {
	//nolint:forbidigo // file is validated
	out, err := getCommandOutput("go", "tool", "cover", fmt.Sprintf("-func=%s", file)) // #nosec G204
	if err != nil {
		return err
	}

	lines := strings.Split(out, "\n")
	var funcs []funcCoverage

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		loc := fields[0]
		name := fields[1]
		covStr := fields[2]

		// Skip "total:" line
		if loc == "total:" {
			continue
		}

		valStr := strings.TrimSuffix(covStr, "%")
		val, err := strconv.ParseFloat(valStr, 64)
		if err != nil {
			continue
		}

		// Only care about < 80% coverage
		if val < 80.0 {
			funcs = append(funcs, funcCoverage{
				Location: loc,
				Name:     name,
				Coverage: val,
			})
		}
	}

	if len(funcs) == 0 {
		return nil
	}

	// Sort by coverage (ascending)
	sort.Slice(funcs, func(i, j int) bool {
		return funcs[i].Coverage < funcs[j].Coverage
	})

	// Take top 10
	topN := 10
	if len(funcs) < topN {
		topN = len(funcs)
	}
	topFuncs := funcs[:topN]

	fmt.Println("Top 10 Functions Missing Tests (< 80%):")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Function\tLocation\tCoverage")
	fmt.Fprintln(w, strings.Repeat("-", 80))

	for _, f := range topFuncs {
		// Location often includes full path, trim to relative if possible
		loc := f.Location
		if idx := strings.Index(loc, "github.com/"); idx != -1 {
			// Try to shorten to repo relative path if possible, but go tool cover output usually has full import path
			// Let's just keep the last 2 segments of path + file:line
			parts := strings.Split(loc, "/")
			if len(parts) > 3 {
				loc = strings.Join(parts[len(parts)-3:], "/")
			}
		}

		color := "\033[31m" // Red
		if f.Coverage >= 50 {
			color = "\033[33m" // Yellow
		}
		reset := "\033[0m"

		name := f.Name
		if len(name) > 30 {
			name = "..." + name[len(name)-27:]
		}

		fmt.Fprintf(w, "%s\t%s\t%s%.1f%%%s\n", name, loc, color, f.Coverage, reset)
	}
	w.Flush()
	fmt.Println()

	return nil
}
