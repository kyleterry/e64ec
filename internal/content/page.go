// Package content loads markdown pages from a directory tree.
package content

import (
	"html/template"
	"path"
	"sort"
	"strings"
	"time"
)

// Frontmatter is the YAML metadata parsed from the top of a markdown file.
type Frontmatter struct {
	Title       string    `yaml:"title"`
	Summary     string    `yaml:"summary"`
	Date        time.Time `yaml:"date"`
	Updated     time.Time `yaml:"updated"`
	Slug        string    `yaml:"slug"`
	Banner      string    `yaml:"banner"`
	Tags        []string  `yaml:"tags"`
	Terms       []string  `yaml:"terms"`
	IncludeTerm string    `yaml:"include_term"`
	Draft       bool      `yaml:"draft"`
	Feed        bool      `yaml:"feed"`
	Lexicon     bool      `yaml:"lexicon"`
	Sort        string    `yaml:"sort"`
}

// Page is a fully loaded and rendered content page.
type Page struct {
	Section     string
	Source      string
	URL         string
	OutputPath  string
	Title       string
	Summary     string
	Banner      string
	Date        time.Time
	Updated     time.Time
	Tags        []string
	Terms       []string
	IncludeTerm string
	Draft       bool
	Feed        bool
	Lexicon     bool
	Sort        string
	Index       bool
	Body        template.HTML
}

// IsHome reports whether this page is the site root.
func (p *Page) IsHome() bool {
	return p.URL == "/"
}

// ModifiedAt returns Updated if set, otherwise Date.
func (p *Page) ModifiedAt() time.Time {
	if !p.Updated.IsZero() {
		return p.Updated
	}
	return p.Date
}

// Pages is a collection of pages with helper methods for common operations.
type Pages []*Page

// Sections returns the unique non-empty section names from the pages,
// sorted alphabetically.
func (ps Pages) Sections() []string {
	seen := map[string]bool{}
	var out []string
	for _, p := range ps {
		if p.Section == "" || seen[p.Section] {
			continue
		}
		seen[p.Section] = true
		out = append(out, p.Section)
	}
	sort.Strings(out)
	return out
}

// HasFeed reports whether any page in the collection has its Feed flag set.
func (ps Pages) HasFeed() bool {
	for _, p := range ps {
		if p.Feed {
			return true
		}
	}
	return false
}

// Sort orders the pages according to the specified mode.
// Supported modes: "date-desc" (default), "date-asc", "title".
// If mode is empty, it uses "date-desc" if any page has a date, else "title".
func (ps Pages) Sort(mode string) {
	if mode == "" {
		anyDate := false
		for _, p := range ps {
			if !p.Date.IsZero() {
				anyDate = true
				break
			}
		}
		if anyDate {
			mode = "date-desc"
		} else {
			mode = "title"
		}
	}

	switch mode {
	case "date-asc":
		sort.Slice(ps, func(i, j int) bool { return ps[i].Date.Before(ps[j].Date) })
	case "title":
		sort.Slice(ps, func(i, j int) bool { return ps[i].Title < ps[j].Title })
	default: // date-desc
		sort.Slice(ps, func(i, j int) bool { return ps[i].Date.After(ps[j].Date) })
	}
}

// Dedupe returns a new collection with duplicate URLs removed and sorted by title.
func (ps Pages) Dedupe() Pages {
	seen := make(map[string]bool)
	out := make(Pages, 0, len(ps))
	for _, p := range ps {
		if seen[p.URL] {
			continue
		}
		seen[p.URL] = true
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Title < out[j].Title })
	return out
}

// ParentURL returns the directory URL containing u, or "" if u is the
// site root. Both u and the return value have trailing slashes.
func ParentURL(u string) string {
	if u == "" || u == "/" {
		return ""
	}

	trimmed := strings.TrimSuffix(u, "/")
	dir := path.Dir(trimmed)
	if dir == "/" || dir == "." {
		return "/"
	}

	return dir + "/"
}

// URLSegment returns the last path segment of a directory URL, used as the
// visible heading on an auto-generated index page.
func URLSegment(u string) string {
	trimmed := strings.Trim(u, "/")
	if trimmed == "" {
		return ""
	}

	segs := strings.Split(trimmed, "/")
	return segs[len(segs)-1]
}

// NormalizeTerms returns a unique, trimmed, non-empty set of terms.
func NormalizeTerms(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, t := range in {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		key := strings.ToLower(t)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, t)
	}
	return out
}
