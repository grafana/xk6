// Package main contains the main entry point for the xk6 CLI.
// This entry point exists for compatibility reasons,
// so that you can still run xk6 with `go run go.k6.io/xk6/cmd/xk6@latest`.
// If you want to use `go run`, use `go run go.k6.io/xk6@latest` instead.
package main

import "go.k6.io/xk6/internal/cmd"

func main() {
	cmd.Execute()
}
