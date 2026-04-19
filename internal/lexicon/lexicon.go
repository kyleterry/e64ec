// Package lexicon builds a term index from pages and auto-links the first
// occurrence of each term in every page.
package lexicon

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"golang.org/x/net/html"

	"go.e64ec.com/e64ec/internal/content"
)

// Entry is a single term with the page that defines it.
type Entry struct {
	Term       string
	Canonical  string
	URL        string
	Source     *content.Page
	References []*content.Page
}

// Lexicon is a collection of Entries keyed by lowercase term.
type Lexicon struct {
	byKey map[string]*Entry
	terms []*Entry
}

// Entries returns entries sorted alphabetically.
func (l *Lexicon) Entries() []*Entry {
	out := make([]*Entry, len(l.terms))
	copy(out, l.terms)
	sort.Slice(out, func(i, j int) bool { return out[i].Canonical < out[j].Canonical })
	return out
}

// TopEntries returns entries sorted by reference count, descending.
func (l *Lexicon) TopEntries(n int) []*Entry {
	out := make([]*Entry, len(l.terms))
	copy(out, l.terms)
	sort.Slice(out, func(i, j int) bool {
		if len(out[i].References) != len(out[j].References) {
			return len(out[i].References) > len(out[j].References)
		}
		return out[i].Canonical < out[j].Canonical
	})
	if n > 0 && n < len(out) {
		return out[:n]
	}
	return out
}

// Link scans rendered HTML and replaces the first occurrence of each term
// with a link to its defining page. The page's own defining term(s) are
// skipped. Anchors, code, and pre nodes are not rewritten.
func (l *Lexicon) Link(page *content.Page) (template.HTML, error) {
	if len(l.terms) == 0 || page.Body == "" {
		return page.Body, nil
	}

	doc, err := html.Parse(strings.NewReader("<root>" + string(page.Body) + "</root>"))
	if err != nil {
		return page.Body, err
	}

	skip := make(map[string]bool)
	for _, t := range page.Terms {
		skip[strings.ToLower(t)] = true
	}
	skip[strings.ToLower(page.Title)] = true

	linked := make(map[string]bool)
	walk(doc, l, skip, linked, page)

	root := findRoot(doc)
	if root == nil {
		return page.Body, nil
	}

	var buf bytes.Buffer
	for c := root.FirstChild; c != nil; c = c.NextSibling {
		if err := html.Render(&buf, c); err != nil {
			return page.Body, err
		}
	}

	return template.HTML(buf.String()), nil
}

// WriteCSV writes the lexicon to path as CSV with columns:
// term, url, source, refs. Entries are sorted alphabetically by term.
func (l *Lexicon) WriteCSV(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}

	if err := l.writeCSV(f); err != nil {
		_ = f.Close()
		return err
	}

	return f.Close()
}

// SitemapNode is a node in the nested sitemap tree.
type SitemapNode struct {
	Title    string
	URL      string
	Children []*SitemapNode
}

// SitemapTree returns a single-rooted tree of all lexicon pages, grouped by
// URL path. Missing ancestor paths are created as placeholder nodes titled
// after their URL segment. Children are sorted alphabetically by title.
func (l *Lexicon) SitemapTree() []*SitemapNode {
	titles := make(map[string]string, len(l.terms))
	for _, e := range l.terms {
		if e.URL == "" {
			continue
		}
		if _, ok := titles[e.URL]; ok {
			continue
		}

		title := e.Canonical
		if e.Source != nil && e.Source.Title != "" {
			title = e.Source.Title
		}
		titles[e.URL] = title
	}

	rootTitle := titles["/"]
	if rootTitle == "" {
		rootTitle = "home"
	}

	root := &SitemapNode{URL: "/", Title: rootTitle}
	nodes := map[string]*SitemapNode{"/": root}

	urls := make([]string, 0, len(titles))
	for u := range titles {
		if u == "/" {
			continue
		}
		urls = append(urls, u)
	}
	sort.Strings(urls)

	for _, u := range urls {
		ensureSitemapPath(u, nodes, titles)
	}

	sortSitemapChildren(root)

	return []*SitemapNode{root}
}

func ensureSitemapPath(u string, nodes map[string]*SitemapNode, titles map[string]string) *SitemapNode {
	if n, ok := nodes[u]; ok {
		return n
	}

	parent := content.ParentURL(u)
	parentNode := ensureSitemapPath(parent, nodes, titles)

	title, ok := titles[u]
	if !ok {
		title = content.URLSegment(u)
	}

	n := &SitemapNode{URL: u, Title: title}
	nodes[u] = n
	parentNode.Children = append(parentNode.Children, n)

	return n
}

func sortSitemapChildren(n *SitemapNode) {
	sort.Slice(n.Children, func(i, j int) bool {
		return n.Children[i].Title < n.Children[j].Title
	})
	for _, c := range n.Children {
		sortSitemapChildren(c)
	}
}

func (l *Lexicon) writeCSV(w io.Writer) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{"term", "url", "source", "refs"}); err != nil {
		return err
	}

	for _, e := range l.Entries() {
		src := ""
		if e.Source != nil {
			src = filepath.ToSlash(e.Source.Source)
		}
		if err := cw.Write([]string{e.Canonical, e.URL, src, fmt.Sprint(len(e.References))}); err != nil {
			return err
		}
	}

	cw.Flush()

	return cw.Error()
}

// New constructs a Lexicon from all pages. All pages contribute their title
// as a term. Term pages also contribute their frontmatter aliases.
func New(pages []*content.Page) (*Lexicon, error) {
	lex := &Lexicon{byKey: map[string]*Entry{}}

	for _, p := range pages {
		terms := collectTerms(p)
		for _, t := range terms {
			key := strings.ToLower(t)
			if existing, dup := lex.byKey[key]; dup {
				return nil, fmt.Errorf("duplicate term %q in %s (also %s)", t, p.Source, existing.Source.Source)
			}
			entry := &Entry{Term: t, Canonical: t, URL: p.URL, Source: p}
			lex.byKey[key] = entry
			lex.terms = append(lex.terms, entry)
		}
	}

	sort.Slice(lex.terms, func(i, j int) bool {
		if len(lex.terms[i].Canonical) != len(lex.terms[j].Canonical) {
			return len(lex.terms[i].Canonical) > len(lex.terms[j].Canonical)
		}
		return lex.terms[i].Canonical < lex.terms[j].Canonical
	})

	return lex, nil
}

func collectTerms(p *content.Page) []string {
	set := []string{p.Title}
	set = append(set, p.Terms...)
	if p.IncludeTerm != "" {
		set = append(set, p.IncludeTerm)
	}
	return content.NormalizeTerms(set)
}

func walk(n *html.Node, lex *Lexicon, skip, linked map[string]bool, page *content.Page) {
	if n.Type == html.ElementNode {
		switch n.Data {
		case "a", "code", "pre", "script", "style", "h1":
			return
		}
	}

	if n.Type == html.TextNode && n.Parent != nil {
		replaced := rewriteText(n, lex, skip, linked, page)
		if replaced {
			return
		}
	}

	for c := n.FirstChild; c != nil; {
		next := c.NextSibling
		walk(c, lex, skip, linked, page)
		c = next
	}
}

func rewriteText(n *html.Node, lex *Lexicon, skip, linked map[string]bool, page *content.Page) bool {
	text := n.Data
	lower := strings.ToLower(text)

	type hit struct {
		start, end int
		entry      *Entry
	}

	var found *hit
	for _, entry := range lex.terms {
		key := strings.ToLower(entry.Canonical)
		if skip[key] || linked[key] || entry.URL == page.URL {
			continue
		}
		re := termRegexp(entry.Canonical)
		loc := re.FindStringIndex(lower)
		if loc == nil {
			continue
		}
		if found == nil || loc[0] < found.start {
			found = &hit{start: loc[0], end: loc[1], entry: entry}
		}
	}

	if found == nil {
		return false
	}

	linked[strings.ToLower(found.entry.Canonical)] = true
	found.entry.References = append(found.entry.References, page)

	before := text[:found.start]
	match := text[found.start:found.end]
	after := text[found.end:]

	parent := n.Parent
	beforeNode := &html.Node{Type: html.TextNode, Data: before}
	anchor := &html.Node{
		Type: html.ElementNode,
		Data: "a",
		Attr: []html.Attribute{
			{Key: "href", Val: found.entry.URL},
			{Key: "class", Val: "lex"},
			{Key: "data-term", Val: found.entry.Canonical},
		},
	}
	anchor.AppendChild(&html.Node{Type: html.TextNode, Data: match})
	afterNode := &html.Node{Type: html.TextNode, Data: after}

	parent.InsertBefore(beforeNode, n)
	parent.InsertBefore(anchor, n)
	parent.InsertBefore(afterNode, n)
	parent.RemoveChild(n)

	walk(afterNode, lex, skip, linked, page)

	return true
}

var termRegexpCache = map[string]*regexp.Regexp{}

func termRegexp(term string) *regexp.Regexp {
	if re, ok := termRegexpCache[term]; ok {
		return re
	}

	quoted := regexp.QuoteMeta(strings.ToLower(term))
	left, right := `\b`, `\b`
	if len(term) > 0 && !isWordRune(rune(term[0])) {
		left = ""
	}
	if len(term) > 0 && !isWordRune(rune(term[len(term)-1])) {
		right = ""
	}
	re := regexp.MustCompile(left + quoted + right)
	termRegexpCache[term] = re

	return re
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

func findRoot(doc *html.Node) *html.Node {
	var f func(*html.Node) *html.Node
	f = func(n *html.Node) *html.Node {
		if n.Type == html.ElementNode && n.Data == "root" {
			return n
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if got := f(c); got != nil {
				return got
			}
		}
		return nil
	}
	return f(doc)
}
