//go:build tools
// +build tools

package tools

// This file ensures tool dependencies are tracked in go.mod
// Tools are not imported in the actual code but are used during development/build

import (
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/pressly/goose/v3/cmd/goose"
	_ "github.com/sqlc-dev/sqlc/cmd/sqlc"
	_ "github.com/swaggo/swag/cmd/swag"
	_ "github.com/vektra/mockery/v2"
	_ "golang.org/x/perf/cmd/benchstat"
)
