// Package generate builds the static site from content into an output directory.
package generate

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/a-h/templ"

	"go.e64ec.com/e64ec/internal/config"
	"go.e64ec.com/e64ec/internal/content"
	"go.e64ec.com/e64ec/internal/lexicon"
	"go.e64ec.com/e64ec/internal/rss"
	"go.e64ec.com/e64ec/internal/ui"
)

// Generator builds the static site from content into an output directory.
type Generator struct {
	cfg *config.Config
}

// Run executes the full site generation pipeline.
func (g *Generator) Run() error {
	rawPages, err := content.NewLoader(g.cfg.ContentDir, g.cfg.PrettyURLs).Load()
	if err != nil {
		return fmt.Errorf("load content: %w", err)
	}
	pages := content.Pages(rawPages)

	lex, err := lexicon.New(pages)
	if err != nil {
		return fmt.Errorf("build lexicon: %w", err)
	}

	site := ui.SiteFrom(g.cfg, pages)

	type renderTask struct {
		p      *content.Page
		linked template.HTML
	}
	tasks := make([]renderTask, 0, len(pages))

	for _, p := range pages {
		linked, err := lex.Link(p)
		if err != nil {
			return fmt.Errorf("link %s: %w", p.Source, err)
		}
		tasks = append(tasks, renderTask{p: p, linked: linked})
	}

	if g.cfg.LexiconFile != "" {
		if err := lex.WriteCSV(g.cfg.LexiconFile); err != nil {
			return fmt.Errorf("write lexicon: %w", err)
		}
	}

	childrenByDir := map[string]content.Pages{}
	for _, p := range pages {
		parent := content.ParentURL(p.URL)
		if parent == "" {
			continue
		}
		childrenByDir[parent] = append(childrenByDir[parent], p)
	}

	for _, task := range tasks {
		p := task.p
		outPath := filepath.Join(g.cfg.SiteDir, filepath.FromSlash(p.OutputPath))

		var backlinks content.Pages
		terms := lex.Entries()
		for _, e := range terms {
			if e.URL == p.URL {
				backlinks = append(backlinks, e.References...)
			}
		}
		backlinks = backlinks.Dedupe()

		var children content.Pages
		if p.Index {
			for _, c := range childrenByDir[p.URL] {
				if c == p {
					continue
				}
				children = append(children, c)
			}
			children.Sort("")
		}

		view := ui.PageView{
			Page:      p,
			Body:      task.linked,
			Backlinks: backlinks,
			Children:  children,
		}
		if err := writeRendered(outPath, ui.Page(site, view)); err != nil {
			return err
		}
	}

	if err := g.writeIndexes(site, pages, lex); err != nil {
		return err
	}

	if err := g.writeFeeds(pages); err != nil {
		return err
	}

	sitemapOut := filepath.Join(g.cfg.SiteDir, "sitemap", "index.html")
	if err := writeRendered(sitemapOut, ui.Sitemap(site, ui.SitemapView{Nodes: lex.SitemapTree()})); err != nil {
		return fmt.Errorf("write sitemap: %w", err)
	}

	if err := copyAssets(g.cfg.AssetsDir, filepath.Join(g.cfg.SiteDir, "assets")); err != nil {
		return err
	}

	fmt.Printf("wrote %d pages to %s\n", len(pages), g.cfg.SiteDir)

	return nil
}

// writeIndexes emits a listing page for every directory in the content tree
// that lacks a user-authored index.md. User-authored index.md with
// `lexicon: true` in frontmatter triggers lexicon rendering instead of the
// normal page body.
func (g *Generator) writeIndexes(site ui.Site, pages content.Pages, lex *lexicon.Lexicon) error {
	userIndexByURL := map[string]*content.Page{}
	childrenByDir := map[string]content.Pages{}
	dirs := map[string]bool{}

	for _, p := range pages {
		if p.Index {
			userIndexByURL[p.URL] = p
			dirs[p.URL] = true
		}

		parent := content.ParentURL(p.URL)
		if parent == "" {
			continue
		}

		dirs[parent] = true
		childrenByDir[parent] = append(childrenByDir[parent], p)
	}

	dirList := make([]string, 0, len(dirs))
	for d := range dirs {
		dirList = append(dirList, d)
	}
	sort.Strings(dirList)

	for _, dir := range dirList {
		if idx, ok := userIndexByURL[dir]; ok {
			if !idx.Lexicon {
				continue
			}

			out := filepath.Join(g.cfg.SiteDir, filepath.FromSlash(idx.OutputPath))
			if err := writeRendered(out, ui.LexiconIndex(site, ui.LexiconView{Page: idx, Entries: lex.TopEntries(0)})); err != nil {
				return err
			}

			continue
		}

		if dir == "/" {
			continue
		}

		items := append(content.Pages(nil), childrenByDir[dir]...)
		items.Sort("")

		heading := content.URLSegment(dir)
		out := filepath.Join(g.cfg.SiteDir, filepath.FromSlash(strings.TrimPrefix(dir, "/")), "index.html")
		if err := writeRendered(out, ui.Index(site, heading, []ui.IndexSection{{Pages: items}})); err != nil {
			return err
		}
	}

	return nil
}

func (g *Generator) writeFeeds(pages content.Pages) error {
	b := rss.NewBuilder(g.cfg)
	for _, f := range rss.DefaultFeeds(g.cfg) {
		b.AddFeed(f)
	}

	for _, feed := range b.Feeds() {
		data, err := b.Build(feed.ID, pages)
		if err != nil {
			return err
		}

		out := filepath.Join(g.cfg.SiteDir, filepath.FromSlash(feed.Path))
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(out, data, 0o644); err != nil {
			return err
		}
	}

	return nil
}

// NewGenerator returns a Generator configured with cfg.
func NewGenerator(cfg *config.Config) *Generator {
	return &Generator{cfg: cfg}
}

func writeRendered(outPath string, c templ.Component) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}

	f, err := os.Create(outPath)
	if err != nil {
		return err
	}

	if err := ui.Render(f, c); err != nil {
		_ = f.Close()
		return err
	}

	return f.Close()
}

func copyAssets(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s: not a directory", src)
	}

	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		return copyFile(p, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}

	return out.Close()
}
