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
	flag.Parse()

	if err := run(cfg); err != nil {
		log.Fatal(err)
	}
}

func run(cfg *config.Config) error {
	pages, err := content.NewLoader(cfg.ContentDir).Load()
	if err != nil {
		return fmt.Errorf("load content: %w", err)
	}

	lex, err := lexicon.New(pages)
	if err != nil {
		return fmt.Errorf("build lexicon: %w", err)
	}

	if cfg.LexiconFile != "" {
		if err := lex.WriteCSV(cfg.LexiconFile); err != nil {
			return fmt.Errorf("write lexicon: %w", err)
		}
	}

	if err := resetSite(cfg.SiteDir); err != nil {
		return err
	}

	site := ui.SiteFrom(cfg)

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

	for _, task := range tasks {
		p := task.p
		outPath := filepath.Join(cfg.SiteDir, filepath.FromSlash(p.OutputPath))

		var backlinks []*content.Page
		terms := lex.Entries()
		for _, e := range terms {
			if e.URL == p.URL {
				backlinks = append(backlinks, e.References...)
			}
		}
		backlinks = dedupePages(backlinks)

		view := ui.PageView{
			Page:      p,
			Body:      task.linked,
			Backlinks: backlinks,
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

	if err := copyAssets(cfg.AssetsDir, filepath.Join(cfg.SiteDir, "assets")); err != nil {
		return err
	}

	fmt.Printf("wrote %d pages to %s\n", len(pages), cfg.SiteDir)

	return nil
}

func writeIndexes(cfg *config.Config, site ui.Site, pages []*content.Page, lex *lexicon.Lexicon) error {
	log := filterKind(pages, content.KindLog)
	sortByDateDesc(log)

	projects := filterKind(pages, content.KindProject)
	sort.Slice(projects, func(i, j int) bool { return projects[i].Title < projects[j].Title })

	if err := writeIndex(cfg, site, "log", "log", filepath.Join(cfg.SiteDir, "log", "index.html"), []ui.IndexSection{
		{Pages: log},
	}); err != nil {
		return err
	}

	if err := writeIndex(cfg, site, "projects", "projects", filepath.Join(cfg.SiteDir, "projects", "index.html"), []ui.IndexSection{
		{Pages: projects},
	}); err != nil {
		return err
	}

	lexOut := filepath.Join(cfg.SiteDir, "terms", "index.html")

	return writeRendered(lexOut, ui.LexiconIndex(site, ui.LexiconView{Entries: lex.TopEntries(0)}))
}

func writeIndex(cfg *config.Config, site ui.Site, heading, _ string, outPath string, sections []ui.IndexSection) error {
	return writeRendered(outPath, ui.Index(site, heading, sections))
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

func writeFeeds(cfg *config.Config, pages []*content.Page) error {
	b := rss.NewBuilder(cfg)
	for _, f := range rss.DefaultFeeds() {
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

func dedupePages(in []*content.Page) []*content.Page {
	seen := make(map[string]bool)
	out := make([]*content.Page, 0, len(in))
	for _, p := range in {
		if seen[p.URL] {
			continue
		}
		seen[p.URL] = true
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Title < out[j].Title })
	return out
}

func filterKind(pages []*content.Page, k content.Kind) []*content.Page {
	out := make([]*content.Page, 0, len(pages))
	for _, p := range pages {
		if p.Kind == k {
			out = append(out, p)
		}
	}
	return out
}

func sortByDateDesc(pages []*content.Page) {
	sort.Slice(pages, func(i, j int) bool {
		return pages[i].Date.After(pages[j].Date)
	})
}
