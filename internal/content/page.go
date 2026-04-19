// Package content loads markdown pages from a directory tree.
package content

import (
	"html/template"
	"time"
)

// Kind classifies a page by its top-level section.
type Kind string

const (
	KindHome    Kind = "home"
	KindLog     Kind = "log"
	KindProject Kind = "project"
	KindTerm    Kind = "term"
	KindOther   Kind = "other"
)

// Frontmatter is the YAML metadata parsed from the top of a markdown file.
type Frontmatter struct {
	Title       string    `yaml:"title"`
	Summary     string    `yaml:"summary"`
	Date        time.Time `yaml:"date"`
	Updated     time.Time `yaml:"updated"`
	Slug        string    `yaml:"slug"`
	Tags        []string  `yaml:"tags"`
	Terms       []string  `yaml:"terms"`
	IncludeTerm string    `yaml:"include_term"`
	Draft       bool      `yaml:"draft"`
	Feed        bool      `yaml:"feed"`
}

// Page is a fully loaded and rendered content page.
type Page struct {
	Kind        Kind
	Source      string
	URL         string
	OutputPath  string
	Title       string
	Summary     string
	Date        time.Time
	Updated     time.Time
	Tags        []string
	Terms       []string
	IncludeTerm string
	Draft       bool
	Feed        bool
	Body        template.HTML
}

// ModifiedAt returns Updated if set, otherwise Date.
func (p *Page) ModifiedAt() time.Time {
	if !p.Updated.IsZero() {
		return p.Updated
	}
	return p.Date
}
