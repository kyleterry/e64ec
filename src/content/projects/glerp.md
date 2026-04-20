---
title: glerp
summary: a scuffed scheme interpreter written in go
tags: [go, scheme]
terms: ["dsl", "scheme"]
---

`glerp` is a scheme interpreter I wrote in Go. I designed it to easily
embeddable in other Go programs. You can use it to write config DSLs or write a
full blown web server. You can create special forms or builtins using Go
functions, or you can write your own syntax using the macro system
(`define-syntax` with `syntax-case` and `syntax-rules`).

Maybe I can use it to generate html from markdown files some day.

I think Scheme is one of my favorite language. I love the simplicity compared to
many of the Lisps. It actually wasn't too bad implementing a good chunk of the
R7RS standard using Go. There's a few features that still need to be implemented
to actually call it a true scheme (continuations??).

Anyway, check it out on [github](https://github.com/kyleterry/glerp).
