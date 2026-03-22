// Package uiassets embeds the compiled SvelteKit dashboard.
// Run `make build-ui` to populate the build/ directory before building the CLI.
package uiassets

import (
	"embed"
	"io/fs"
)

//go:embed all:build
var embeddedFS embed.FS

// FS is the root of the embedded UI build output.
var FS, _ = fs.Sub(embeddedFS, "build")
