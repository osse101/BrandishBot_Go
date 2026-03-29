package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

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

		// Block shell redirection and pipes to prevent potential downstream shell injection.
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
	return runCommandExt(nil, os.Stdout, os.Stderr, name, args...)
}

// runCommandWithStdin runs a command with the provided stdin
func runCommandWithStdin(stdin io.Reader, name string, args ...string) error {
	return runCommandExt(stdin, os.Stdout, os.Stderr, name, args...)
}

// runCommandToFile runs a command and redirects stdout to a file
func runCommandToFile(stdout io.Writer, name string, args ...string) error {
	return runCommandExt(nil, stdout, os.Stderr, name, args...)
}

// runCommandAsync starts a command and returns the *exec.Cmd for manual management
func runCommandAsync(name string, args ...string) (*exec.Cmd, error) {
	if err := checkHostile(append([]string{name}, args...)...); err != nil {
		return nil, err
	}
	// #nosec G204 - Generic command wrapper
	cmd := exec.Command(name, args...)
	return cmd, cmd.Start()
}

// runCommandWithStdoutPipe starts a command and returns its stdout pipe and the *exec.Cmd
func runCommandWithStdoutPipe(name string, args ...string) (io.ReadCloser, *exec.Cmd, error) {
	if err := checkHostile(append([]string{name}, args...)...); err != nil {
		return nil, nil, err
	}
	// #nosec G204 - Generic command wrapper
	cmd := exec.Command(name, args...)
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}
	return pipe, cmd, cmd.Start()
}

// runCommandExt is the base for other wrappers, allowing full IO control while keeping security checks
func runCommandExt(stdin io.Reader, stdout io.Writer, stderr io.Writer, name string, args ...string) error {
	if err := checkHostile(append([]string{name}, args...)...); err != nil {
		return err
	}
	// #nosec G204 - Generic command wrapper
	cmd := exec.Command(name, args...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}
