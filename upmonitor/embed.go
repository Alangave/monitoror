// Package upmonitor embeds the built Node.js UI (web/dist) into the Go binary.
package upmonitor

import (
	"embed"
	"io/fs"
)

//go:embed all:web/dist
var distFS embed.FS

// Dist returns the UI filesystem rooted at web/dist.
func Dist() fs.FS {
	sub, err := fs.Sub(distFS, "web/dist")
	if err != nil {
		panic(err)
	}
	return sub
}
