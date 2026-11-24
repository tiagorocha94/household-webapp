// Package web holds the built frontend, embedded in the server binary so that
// one file serves both the API and the UI.
package web

import (
	"embed"
	"io/fs"
)

// dist is web/dist, written by "npm run build" (make run does it). A fresh
// checkout holds only a placeholder, so the Go code always compiles; the
// server then serves the API alone until the frontend is built.
//
//go:embed all:dist
var dist embed.FS

// Dist returns the built frontend, rooted at its index.html.
func Dist() fs.FS {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		// The pattern above guarantees the folder exists.
		panic(err)
	}
	return sub
}
