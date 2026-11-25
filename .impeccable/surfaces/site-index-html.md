---
version: 1
slug: "site-index-html"
primary_target: "site/index.html"
related_targets: []
---

## Scope

`site/index.html`, the dirstato landing page on GitHub Pages
(expected at https://lemiorhanergin.com/dirstato/, via the user's custom
domain). Visitor mode: Persuade.

## Audience, job, action

- Audience: macOS developers who live in the terminal and know `du`/`ncdu`.
- Job: decide within seconds whether to install.
- Action: copy `brew install lemiorhan/dirstato/dirstato` (go install and
  source as alternates).
- Proof: a real scan of `/opt/homebrew/Cellar/go/1.27.4/libexec` (captured
  2026-11-25 with the repo's scanner, 277084 KiB, `du -skx` 277084); the eval
  test names; the one dated performance measurement from the README.
- Constraints: no invented testimonials, counts or benchmarks; demo stays
  monochrome like the real TUI; relative URLs (served under `/dirstato/`).

## Direction contract

THESIS: The page is a dirstato session the visitor can drive, on one character-cell surface. It refuses the CLI-page default: a dark hero, a floating terminal-window screenshot with a glow, and a grid of feature cards.

OWN-WORLD: A terminal profile that follows the system appearance: near-white ground with ink in light, ink ground with soft white in dark, faint ink for secondary. One accent, the logo's coral, reserved for the cursor: prompt block cursor, focus, text selection. Box-drawing rules divide the surface; block bars, ▸▾ markers, reverse-video selection. Monaspace Xenon heavy and wide for headlines, Monaspace Neon for everything else. No cards, shadows, glows or rounded boxes.

STORY: The visitor reads the headline and the brew line, watches a real scan resolve into the tree, drives it, sees the root total agree with `du -skx`, reads the counting rules pinned by tests and the dated speed measurement, and copies the install line, offered again at the close.

FIRST VIEWPORT: Line one is an app-style header line as nav, a rule under it. The headline in Xenon at display scale, one subline, then the `$ brew install …` prompt with block cursor and copy control, the primary action. Below, the live session starts above the fold: scan progress, then rows. On wide screens a note column beside it explains the row under the cursor. Phones drop date, then count, like the real app.

FORM: Text Mode, position 7 of 7 on my ordered list (assigned by the roll). seed:5d66634b8960b571. Raises: cross-check (six-pack), legend (orienteering), one surface (ebru), scale courage (flyer wall), one lit run (neon), proximity reveal (understory). Signature interaction: the app's keymap drives the demo (j k ↑↓ → ← Tab s m c r ?), with click and touch equivalents. Motion grammar: terminal-native, instant redraws, a ticking scan counter, one blinking block cursor; reduced motion skips the scan.

FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance

## Memorable moment

Pressing a key on a web page and watching a real dirstato tree respond, then seeing 277084 next to 277084.

## Unresolved

- Whether a dedicated OG image beyond the logo is wanted.
- README/CHANGELOG links to the site: offered, not done.
