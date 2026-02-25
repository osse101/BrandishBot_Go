package handler

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Initialize validator once for all tests in this package to avoid race conditions
	InitValidator()
	os.Exit(m.Run())
}
