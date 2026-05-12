package webui

import (
	"embed"
	"io/fs"
)

//go:embed dist
var distFS embed.FS

// DistFS returns the embedded dist directory as an fs.FS,
// with the "dist/" prefix stripped so index.html, assets/* etc.
// are at the root of the returned filesystem.
func DistFS() fs.FS {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic(err)
	}
	return sub
}
