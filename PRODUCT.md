# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Stack

Static HTML, CSS and vanilla JS in `site/`, no framework and no build step.
A GitHub Actions workflow deploys `site/` to GitHub Pages; `docs/` (Turkish
spec and plan) is not published.

## Users

Developers on macOS who live in the terminal and already know `du`, `ncdu`,
`dust` or `gdu`. Their disk is full or filling, and they want to find what is
taking the space quickly, without a GUI and without a tool that might delete
something. They arrive from GitHub, Hacker News or Reddit and decide within
seconds whether to install.

## Product Purpose

dirstato scans a folder in parallel and lets you walk the result in the
terminal, as an expandable tree or folder by folder, with size, share of the
parent, a bar, recursive file count and newest modification date per row.
Success (inferred, unconfirmed): a developer installs it with one command and
trusts the numbers it shows.

## Positioning

Its numbers match `du -skx`: allocated disk blocks, hard links counted once,
symlinks not followed, one file system. This is pinned by tests that compare
the root total with `/usr/bin/du` on fixture trees (hard links, symlinks,
sparse files, unreadable folders, a mounted disk image). It is also strictly
read-only: there is no delete key.

## Operating Context

Run as `dirstato [path] [--min SIZE]` in a terminal on macOS. Keyboard driven:
arrows or `hjkl`, `Tab` switches tree and drill-down, `s` sort, `m` minimum
size, `c` copy path (`pbcopy`), `r` rescan in place, `?` help, `q` quit.

## Capabilities and Constraints

- macOS only (macOS system calls, `pbcopy`).
- Install: `brew install lemiorhan/dirstato/dirstato` (tap builds from
  source), `go install github.com/lemiorhan/dirstato@latest` (Go 1.27+), or
  build from source.
- Items under `--min` fold into one `… N small items` row per folder;
  unreadable folders are marked `!`.
- Narrow terminals drop the date column first, then the count.
- Licensed Apache 2.0. Current release 0.1.0 (2026-11-25).

## Brand Commitments

- Name is lowercase: `dirstato`.
- Logo: `docs/assets/logo.svg` is an asset the site must show (a dark tile
  with a tree outline and four rounded bars in `#ff6b4a`, `#ffb020`,
  `#2dd4bf`, `#60a5fa`). Open: whether its palette or dark tile binds the
  site's world.
- Inferred from the README, unconfirmed: voice is plain, factual, no hype
  ("dirstato tells you where the space went; you decide what to do about
  it"), and site copy is English like the README and the interface.

## Evidence on Hand

- Real TUI output: the README's scan of a Go installation with `--min 4M`.
- One performance measurement (author's machine, 2026-11-24, `~/Code`):
  dirstato 11.6 s vs `du -skx` 46.5 s, 1.57 million nodes, 183 MB heap after
  GC, peak RSS 288 MB, root size identical to `du`. It must always be shown
  with its date and conditions.
- Eval tests: `TestEvalScanMatchesDu`, `TestEvalRescanMatchesDu`,
  `TestEvalMountBoundary` in `internal/scan/scan_eval_test.go`.
- Absent, and not to be fabricated: testimonials, user or download counts,
  star counts, press, benchmarks against `ncdu`, `dust`, `gdu` or any GUI app.

## Product Principles

Inferred from the README and the interview answers, unconfirmed:

- Trust over flash: every number shown can be traced to `du` or a test.
- Read-only by design: the tool reports, the user decides.
- One command to start: installation and first run stay a single line each.
- Keyboard first: everything is reachable from a key, and the keys are few.
