// Package config holds site-wide configuration.
package config

import "time"

// Config is the static-site build configuration.
type Config struct {
	BaseURL     string
	Title       string
	Description string
	Author      string
	Language    string
	ContentDir  string
	AssetsDir   string
	SiteDir     string
	LexiconFile string
	BuildTime   time.Time
}

// Default returns the baked-in configuration for e64ec.com.
func Default() *Config {
	return &Config{
		BaseURL:     "https://e64ec.com",
		Title:       "e64ec log",
		Description: "Personal wiki and log.",
		Author:      "",
		Language:    "en",
		ContentDir:  "content",
		AssetsDir:   "assets",
		SiteDir:     "site/public",
		LexiconFile: "lexicon",
		BuildTime:   time.Now().UTC(),
	}
}
