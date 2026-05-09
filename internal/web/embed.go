// Package web embeds the Vite SPA build output.
//
// The bundle is built by `npm run build` in the repo's web/ directory;
// the resulting web/dist tree is embedded here at compile time. If the
// dist tree is empty, DistFS returns nil and the server falls back to a
// helpful error message.
package web

import (
	"embed"
	"io/fs"
	"mime"
)

func init() {
	// Go's default MIME table maps neither `.webmanifest` nor the
	// recommended type, so http.FileServer would otherwise serve the
	// PWA manifest as application/octet-stream and browsers would skip
	// it.
	_ = mime.AddExtensionType(".webmanifest", "application/manifest+json")
}

//go:embed all:dist
var distFS embed.FS

// DistFS returns the embedded SPA bundle rooted at "dist/", or nil if
// no build output is present.
func DistFS() fs.FS {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil
	}
	entries, err := fs.ReadDir(sub, ".")
	if err != nil || len(entries) == 0 {
		return nil
	}
	// Only the .gitkeep marker → treat as empty.
	if len(entries) == 1 && entries[0].Name() == ".gitkeep" {
		return nil
	}
	return sub
}
