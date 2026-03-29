package main

import (
	"fmt"
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
