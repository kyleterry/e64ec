package content

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// Loader walks a content directory and produces Pages.
type Loader struct {
	root     string
	markdown goldmark.Markdown
}

// Load reads and returns all pages under l.root sorted by URL.
func (l *Loader) Load() ([]*Page, error) {
	var pages []*Page

	err := filepath.WalkDir(l.root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !isContentFile(p) {
			return nil
		}

		page, err := l.loadFile(p)
		if err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
		if page == nil {
			return nil
		}

		pages = append(pages, page)
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(pages, func(i, j int) bool { return pages[i].URL < pages[j].URL })

	return pages, nil
}

func (l *Loader) loadFile(abs string) (*Page, error) {
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}

	rel, err := filepath.Rel(l.root, abs)
	if err != nil {
		return nil, err
	}
	rel = filepath.ToSlash(rel)

	fmRaw, body, err := splitFrontmatter(data)
	if err != nil {
		return nil, err
	}

	fm, err := parseFrontmatter(fmRaw)
	if err != nil {
		return nil, err
	}

	if fm.Draft {
		return nil, nil
	}

	page := &Page{
		Source:      abs,
		Title:       fm.Title,
		Summary:     fm.Summary,
		Date:        fm.Date,
		Updated:     fm.Updated,
		Tags:        fm.Tags,
		Terms:       normalizeTerms(fm.Terms),
		IncludeTerm: fm.IncludeTerm,
		Draft:       fm.Draft,
		Feed:        fm.Feed,
	}

	page.Kind = classify(rel)
	page.URL, page.OutputPath = routeFor(rel, fm.Slug)

	ext := strings.ToLower(filepath.Ext(abs))
	switch ext {
	case ".md", ".markdown":
		var buf bytes.Buffer
		if err := l.markdown.Convert(body, &buf); err != nil {
			return nil, fmt.Errorf("markdown: %w", err)
		}
		page.Body = template.HTML(buf.String())
	case ".html", ".htm":
		page.Body = template.HTML(body)
	default:
		return nil, fmt.Errorf("unsupported extension %q", ext)
	}

	if page.Title == "" {
		page.Title = deriveTitle(rel)
	}

	return page, nil
}

func isContentFile(p string) bool {
	switch strings.ToLower(filepath.Ext(p)) {
	case ".md", ".markdown", ".html", ".htm":
		return true
	}
	return false
}

func classify(rel string) Kind {
	top, _, _ := strings.Cut(rel, "/")
	switch top {
	case "log":
		return KindLog
	case "projects":
		return KindProject
	case "terms":
		return KindTerm
	case rel:
		if rel == "index.md" || rel == "index.html" {
			return KindHome
		}
		return KindOther
	}
	return KindOther
}

func routeFor(rel, slug string) (url, outputPath string) {
	ext := filepath.Ext(rel)
	withoutExt := strings.TrimSuffix(rel, ext)
	base := path.Base(withoutExt)
	dir := path.Dir(withoutExt)

	if slug != "" {
		base = slug
	}

	if base == "index" {
		if dir == "." || dir == "" {
			return "/", "index.html"
		}
		return "/" + dir + "/", path.Join(dir, "index.html")
	}

	urlPath := "/"
	if dir != "." && dir != "" {
		urlPath += dir + "/"
	}
	urlPath += base + "/"

	out := path.Join(strings.TrimPrefix(urlPath, "/"), "index.html")
	return urlPath, out
}

func deriveTitle(rel string) string {
	ext := filepath.Ext(rel)
	base := strings.TrimSuffix(filepath.Base(rel), ext)
	base = strings.ReplaceAll(base, "-", " ")
	base = strings.ReplaceAll(base, "_", " ")
	if base == "" {
		return rel
	}
	return strings.ToUpper(base[:1]) + base[1:]
}

func normalizeTerms(in []string) []string {
	out := make([]string, 0, len(in))
	seen := make(map[string]bool, len(in))
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

// NewLoader builds a content loader rooted at root.
func NewLoader(root string) *Loader {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Footnote, extension.Typographer),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(html.WithUnsafe()),
	)

	return &Loader{root: root, markdown: md}
}
