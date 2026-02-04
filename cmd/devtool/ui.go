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

func getCommandOutput(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// runCommand runs a command silently
func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

// runCommandVerbose runs a command and pipes output to stdout/stderr
func runCommandVerbose(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
