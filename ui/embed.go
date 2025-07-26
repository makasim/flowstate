package ui

import (
	"embed"
	"io/fs"
)

//go:embed dist
var publicFS embed.FS

func PublicFS() fs.FS {
	fsys, _ := fs.Sub(publicFS, "dist")
	return fsys
}
