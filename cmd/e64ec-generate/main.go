// Command e64ec-generate builds the static site from content/ into site/.
package main

import (
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
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

func main() {
	cfg := config.Default()

	flag.StringVar(&cfg.ContentDir, "content", cfg.ContentDir, "content source directory")
	flag.StringVar(&cfg.AssetsDir, "assets", cfg.AssetsDir, "assets source directory")
	flag.StringVar(&cfg.SiteDir, "out", cfg.SiteDir, "output directory")
	flag.StringVar(&cfg.LexiconFile, "lexicon", cfg.LexiconFile, "lexicon CSV output path")
	flag.StringVar(&cfg.BaseURL, "base-url", cfg.BaseURL, "canonical base URL")
	flag.BoolVar(&cfg.PrettyURLs, "pretty-urls", cfg.PrettyURLs, "use directory-based (pretty) URLs")
	flag.Parse()

	if err := run(cfg); err != nil {
		log.Fatal(err)
	}
}

func run(cfg *config.Config) error {
	rawPages, err := content.NewLoader(cfg.ContentDir, cfg.PrettyURLs).Load()
	if err != nil {
		return fmt.Errorf("load content: %w", err)
	}
	pages := content.Pages(rawPages)

	lex, err := lexicon.New(pages)
	if err != nil {
		return fmt.Errorf("build lexicon: %w", err)
	}

	site := ui.SiteFrom(cfg, pages)

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

	if cfg.LexiconFile != "" {
		if err := lex.WriteCSV(cfg.LexiconFile); err != nil {
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
		outPath := filepath.Join(cfg.SiteDir, filepath.FromSlash(p.OutputPath))

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

	if err := writeIndexes(cfg, site, pages, lex); err != nil {
		return err
	}

	if err := writeFeeds(cfg, pages); err != nil {
		return err
	}

	sitemapOut := filepath.Join(cfg.SiteDir, "sitemap", "index.html")
	if err := writeRendered(sitemapOut, ui.Sitemap(site, ui.SitemapView{Nodes: lex.SitemapTree()})); err != nil {
		return fmt.Errorf("write sitemap: %w", err)
	}

	if err := copyAssets(cfg.AssetsDir, filepath.Join(cfg.SiteDir, "assets")); err != nil {
		return err
	}

	fmt.Printf("wrote %d pages to %s\n", len(pages), cfg.SiteDir)

	return nil
}

// writeIndexes emits a listing page for every directory in the content tree
// that lacks a user-authored index.md. User-authored index.md with
// `lexicon: true` in frontmatter triggers lexicon rendering instead of the
// normal page body.
func writeIndexes(cfg *config.Config, site ui.Site, pages content.Pages, lex *lexicon.Lexicon) error {
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

			out := filepath.Join(cfg.SiteDir, filepath.FromSlash(idx.OutputPath))
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
		out := filepath.Join(cfg.SiteDir, filepath.FromSlash(strings.TrimPrefix(dir, "/")), "index.html")
		if err := writeRendered(out, ui.Index(site, heading, []ui.IndexSection{{Pages: items}})); err != nil {
			return err
		}
	}

	return nil
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

func writeFeeds(cfg *config.Config, pages content.Pages) error {
	b := rss.NewBuilder(cfg)
	for _, f := range rss.DefaultFeeds(cfg) {
		b.AddFeed(f)
	}

	for _, feed := range b.Feeds() {
		data, err := b.Build(feed.ID, pages)
		if err != nil {
			return err
		}

		out := filepath.Join(cfg.SiteDir, filepath.FromSlash(feed.Path))
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(out, data, 0o644); err != nil {
			return err
		}
	}

	return nil
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

func resetSite(dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	return os.MkdirAll(dir, 0o755)
}
