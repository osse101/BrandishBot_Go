package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	colorGreen  = "\033[0;32m"
	colorRed    = "\033[0;31m"
	colorYellow = "\033[1;33m"
	colorBlue   = "\033[0;34m"
	colorReset  = "\033[0m"
)

// UI helpers

func PrintInfo(format string, a ...interface{}) {
	fmt.Printf(colorBlue+"ℹ "+format+colorReset+"\n", a...)
}

func PrintSuccess(format string, a ...interface{}) {
	fmt.Printf(colorGreen+"✓ "+format+colorReset+"\n", a...)
}

func PrintWarning(format string, a ...interface{}) {
	fmt.Printf(colorYellow+"⚠ "+format+colorReset+"\n", a...)
}

func PrintError(format string, a ...interface{}) {
	fmt.Printf(colorRed+"✗ "+format+colorReset+"\n", a...)
}

func PrintHeader(title string) {
	fmt.Printf("\n"+colorYellow+"=== %s ==="+colorReset+"\n", title)
}

// Command execution helpers

// checkHostile checks for potentially dangerous strings in command arguments.
// It focuses on common shell injection patterns while being permissive enough
// for valid data like URLs (containing '&') and SQL (containing ';').
func checkHostile(inputs ...string) error {
	for _, s := range inputs {
		// Newlines/CR are always suspicious in command args as they can split commands
		if strings.ContainsAny(s, "\n\r") {
			return fmt.Errorf("hostile input detected: newlines or carriage returns")
		}

		// Null bytes are often used in exploit payloads
		if strings.Contains(s, "\x00") {
			return fmt.Errorf("hostile input detected: null byte")
		}

		// Shell redirection, pipes, and command substitution patterns.
		// These are blocked because even if exec.Command is generally safe,
		// these arguments might eventually be passed to a shell-executing process.
		dangerousPats := []string{"|", "`", "$(", "&&", "||", ">", "<"}
		for _, p := range dangerousPats {
			if strings.Contains(s, p) {
				return fmt.Errorf("hostile input detected: pattern %q in %q", p, s)
			}
		}
	}
	return nil
}

func getCommandOutput(name string, args ...string) (string, error) {
	if err := checkHostile(append([]string{name}, args...)...); err != nil {
		return "", err
	}
	// #nosec G204 - Generic command wrapper
	cmd := exec.Command(name, args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// runCommand runs a command silently
func runCommand(name string, args ...string) error {
	if err := checkHostile(append([]string{name}, args...)...); err != nil {
		return err
	}
	// #nosec G204 - Generic command wrapper
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

// runCommandVerbose runs a command and pipes output to stdout/stderr
func runCommandVerbose(name string, args ...string) error {
	if err := checkHostile(append([]string{name}, args...)...); err != nil {
		return err
	}
	// #nosec G204 - Generic command wrapper
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
