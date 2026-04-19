package ui

import (
	"context"
	"io"

	"github.com/a-h/templ"
)

// Render runs a templ component against w with a background context.
func Render(w io.Writer, c templ.Component) error {
	return c.Render(context.Background(), w)
}
