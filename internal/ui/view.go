// Package ui holds templ-rendered layouts and view models for the site.
package ui

import (
	"html/template"
	"strings"
	"time"

	"go.e64ec.com/e64ec/internal/config"
	"go.e64ec.com/e64ec/internal/content"
	"go.e64ec.com/e64ec/internal/lexicon"
)

// Site is the global view-model passed to every layout.
type Site struct {
	Title       string
	Description string
	Categories  []string
	BaseURL     string
	Author      string
	Language    string
	Now         time.Time
	Nav         []NavLink
}

func (s *Site) Keywords() string {
	return strings.Join(s.Categories, ",")
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
	Children  []*content.Page
}

// IndexSection is a titled list of pages rendered on an index page.
type IndexSection struct {
	Title string
	Pages []*content.Page
}

// LexiconView is the view-model for the lexicon index page.
type LexiconView struct {
	Page    *content.Page
	Entries []*lexicon.Entry
}

// SitemapView is the view-model for the HTML sitemap page.
type SitemapView struct {
	Nodes []*lexicon.SitemapNode
}

// SiteFrom builds a Site from config and a derived nav. Nav entries are
// generated from the unique top-level sections discovered in pages,
// in alphabetical order with home first and feed last.
func SiteFrom(cfg *config.Config, pages content.Pages) Site {
	nav := []NavLink{{Title: "home", URL: "/"}}

	for _, s := range pages.Sections() {
		nav = append(nav, NavLink{Title: s, URL: "/" + s + "/"})
	}

	if pages.HasFeed() {
		nav = append(nav, NavLink{Title: "feed", URL: "/feed.xml"})
	}

	return Site{
		Title:       cfg.Title,
		Description: cfg.Description,
		Categories:  cfg.Categories,
		BaseURL:     cfg.BaseURL,
		Author:      cfg.Author,
		Language:    cfg.Language,
		Now:         cfg.BuildTime,
		Nav:         nav,
	}
}
