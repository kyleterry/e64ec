// Command e64ec-generate builds the static site from content/ into site/.
package main

import (
	"flag"
	"log"

	"go.e64ec.com/e64ec/internal/config"
	"go.e64ec.com/e64ec/internal/generate"
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

	if err := generate.NewGenerator(cfg).Run(); err != nil {
		log.Fatal(err)
	}
}
