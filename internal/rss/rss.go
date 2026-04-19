// Package rss builds RSS 2.0 feeds from content pages.
package rss

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"sort"
	"strings"
	"time"

	"go.e64ec.com/e64ec/internal/config"
	"go.e64ec.com/e64ec/internal/content"
)

// Filter reports whether a page should appear in a feed.
type Filter func(*content.Page) bool

// Feed describes a single RSS feed.
type Feed struct {
	ID          string
	Title       string
	Description string
	Path        string
	Filter      Filter
}

// Builder constructs RSS XML for configured feeds.
type Builder struct {
	cfg   *config.Config
	feeds []Feed
}

// AddFeed registers a new feed.
func (b *Builder) AddFeed(f Feed) {
	b.feeds = append(b.feeds, f)
}

// Feeds returns the registered feeds.
func (b *Builder) Feeds() []Feed {
	out := make([]Feed, len(b.feeds))
	copy(out, b.feeds)
	return out
}

// Build returns XML for a single feed selected by id.
func (b *Builder) Build(id string, pages content.Pages) ([]byte, error) {
	var feed *Feed
	for i := range b.feeds {
		if b.feeds[i].ID == id {
			feed = &b.feeds[i]
			break
		}
	}
	if feed == nil {
		return nil, fmt.Errorf("unknown feed %q", id)
	}

	items := make([]rssItem, 0, len(pages))
	for _, p := range pages {
		if feed.Filter != nil && !feed.Filter(p) {
			continue
		}
		if p.Date.IsZero() {
			continue
		}
		items = append(items, rssItem{
			Title:       p.Title,
			Link:        joinURL(b.cfg.BaseURL, p.URL),
			GUID:        guid{Value: joinURL(b.cfg.BaseURL, p.URL), IsPermaLink: "true"},
			PubDate:     p.Date.UTC().Format(time.RFC1123Z),
			Description: strings.TrimSpace(p.Summary),
		})
	}

	sort.Slice(items, func(i, j int) bool { return items[i].PubDate > items[j].PubDate })

	doc := rssRoot{
		Version: "2.0",
		Channel: rssChannel{
			Title:         feed.Title,
			Link:          b.cfg.BaseURL,
			Description:   feed.Description,
			Language:      b.cfg.Language,
			Categories:    b.cfg.Categories,
			LastBuildDate: b.cfg.BuildTime.Format(time.RFC1123Z),
			Generator:     "e64ec",
			Items:         items,
		},
	}

	var buf bytes.Buffer
	buf.WriteString(xml.Header)

	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// DefaultFeeds returns the standard feed set for cfg. Pages opt in via
// `feed: true` in frontmatter, or by living in any cfg.FeedSections section.
func DefaultFeeds(cfg *config.Config) []Feed {
	included := make(map[string]bool, len(cfg.FeedSections))
	for _, s := range cfg.FeedSections {
		included[s] = true
	}

	return []Feed{
		{
			ID:          "main",
			Title:       cfg.Title,
			Description: "Updates from " + cfg.BaseURL,
			Path:        "feed.xml",
			Filter: func(p *content.Page) bool {
				if p.Index {
					return false
				}
				if p.Feed {
					return true
				}
				return included[p.Section]
			},
		},
	}
}

// NewBuilder constructs a Builder for cfg with no feeds registered.
func NewBuilder(cfg *config.Config) *Builder {
	return &Builder{cfg: cfg}
}

type rssRoot struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title         string    `xml:"title"`
	Link          string    `xml:"link"`
	Description   string    `xml:"description"`
	Language      string    `xml:"language,omitempty"`
	LastBuildDate string    `xml:"lastBuildDate,omitempty"`
	Generator     string    `xml:"generator,omitempty"`
	Categories    []string  `xml:"category,omitempty"`
	Items         []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	GUID        guid   `xml:"guid"`
	PubDate     string `xml:"pubDate"`
	Description string `xml:"description,omitempty"`
}

type guid struct {
	Value       string `xml:",chardata"`
	IsPermaLink string `xml:"isPermaLink,attr,omitempty"`
}

func joinURL(base, pathPart string) string {
	base = strings.TrimRight(base, "/")
	if !strings.HasPrefix(pathPart, "/") {
		pathPart = "/" + pathPart
	}
	return base + pathPart
}
