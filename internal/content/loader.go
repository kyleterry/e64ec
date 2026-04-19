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
		Banner:      fm.Banner,
		Date:        fm.Date,
		Updated:     fm.Updated,
		Tags:        fm.Tags,
		Terms:       NormalizeTerms(fm.Terms),
		IncludeTerm: fm.IncludeTerm,
		Draft:       fm.Draft,
		Feed:        fm.Feed,
		Lexicon:     fm.Lexicon,
		Sort:        fm.Sort,
	}

	page.Section = sectionFor(rel)
	page.Index = isIndexFile(rel)
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

// sectionFor returns the top-level directory for rel, or "" for root files.
func sectionFor(rel string) string {
	top, rest, found := strings.Cut(rel, "/")
	if !found || rest == "" {
		return ""
	}
	return top
}

// isIndexFile reports whether rel is a directory index (index.md or index.html).
func isIndexFile(rel string) bool {
	base := strings.ToLower(path.Base(rel))
	return base == "index.md" || base == "index.html" || base == "index.htm" || base == "index.markdown"
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
	withoutExt := strings.TrimSuffix(rel, ext)
	base := path.Base(withoutExt)
	if base == "index" {
		parent := path.Base(path.Dir(withoutExt))
		if parent != "." && parent != "/" && parent != "" {
			return parent
		}
	}

	return base
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
