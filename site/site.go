// Package site embeds the generated static site for serving.
package site

import (
	"embed"
	"io/fs"
)

//go:embed all:public
var embedded embed.FS

// FS returns the generated site rooted at its top level.
func FS() (fs.FS, error) {
	return fs.Sub(embedded, "public")
}
