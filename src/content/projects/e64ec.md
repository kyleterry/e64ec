---
title: e64ec
summary: This site, and its static-site generator.
tags: [go, templ]
include_term: e64ec
terms: ["markdown files"]
---

`e64ec` is the static-site generator powering this site. It turns a tree of
markdown files under `content/` into a site under `site/public/`.

Design notes:

- Written in Go, templated with templ.
- Uses a lexicon to auto-link defined terms.
- Produces a single RSS feed today; the feed generator is extensible.
