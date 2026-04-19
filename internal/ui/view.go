// Package ui holds templ-rendered layouts and view models for the site.
package ui

import (
	"html/template"
	"time"

	"go.e64ec.com/e64ec/internal/config"
	"go.e64ec.com/e64ec/internal/content"
	"go.e64ec.com/e64ec/internal/lexicon"
)

// Site is the global view-model passed to every layout.
type Site struct {
	Title       string
	Description string
	BaseURL     string
	Author      string
	Language    string
	Now         time.Time
	Nav         []NavLink
}

// NavLink is a primary site nav entry.
type NavLink struct {
	Title string
	URL   string
}

// PageView wraps a content.Page with rendered body for templates.
type PageView struct {
	Page      *content.Page
	Body      template.HTML
	Backlinks []*content.Page
}

// IndexSection is a titled list of pages rendered on an index page.
type IndexSection struct {
	Title string
	Kind  content.Kind
	Pages []*content.Page
}

// LexiconView is the view-model for the lexicon index page.
type LexiconView struct {
	Entries []*lexicon.Entry
}

// SiteFrom builds a Site from config and a default nav.
func SiteFrom(cfg *config.Config) Site {
	return Site{
		Title:       cfg.Title,
		Description: cfg.Description,
		BaseURL:     cfg.BaseURL,
		Author:      cfg.Author,
		Language:    cfg.Language,
		Now:         cfg.BuildTime,
		Nav: []NavLink{
			{Title: "home", URL: "/"},
			{Title: "log", URL: "/log/"},
			{Title: "projects", URL: "/projects/"},
			{Title: "terms", URL: "/terms/"},
			{Title: "feed", URL: "/feed.xml"},
		},
	}
}
