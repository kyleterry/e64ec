---
title: glerp
summary: a scuffed scheme interpreter written in go
date: 2026-04-18
tags: [go, scheme]
---

`glerp` is a scheme interpreter I wrote in Go. I designed it to easily
embeddable in other Go programs. You can use it to write config DSLs or write a
full blown web server. You can create special forms or builtins using Go
functions, or you can write your own syntax using the macro system
(`define-syntax` with `syntax-case` and `syntax-rule`).
