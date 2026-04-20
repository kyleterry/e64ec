---
title: booksmk
summary: a scuffed bookmarking and feed reader service written in go
tags: [go, postgres]
terms: ["bookmark", "feed reader"]
---

`booksmk` is another nutty project from me. It is a bookmarking service that
morphed into a feed reader. It has a lot of handy features and just gives you a
nice simple interface to use. When you add a URL, it will try its hardest to
find a feed link in both the page and the site root. If it finds one, it will
offer it up to the user with a "subscribe to feed" button. It also searches a
few services like Hacker News and Reddit for discussions about the URL (if
another user submitted it).

Check it out on [github](https://github.com/kyleterry/booksmk).
