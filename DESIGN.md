---
name: dirstato
description: A read-only disk usage explorer for the macOS terminal, presented as one character-cell surface the visitor can drive.
colors:
  paper: "#fbfbfa"
  ink: "#141518"
  faint: "#60666c"
  rule: "#c6cace"
  focus: "#d4401c"
  paper-dark: "#0f1012"
  ink-dark: "#e6e7e9"
  faint-dark: "#8f959b"
  rule-dark: "#3a3e43"
  focus-dark: "#ff7d5f"
  coral: "#ff6b4a"
  selection-text: "#141518"
typography:
  display:
    fontFamily: "Dirstato Display, Dirstato Text, ui-monospace, Menlo, monospace"
    fontSize: "clamp(2.3rem, 5.2vw, 4.75rem)"
    fontWeight: 800
    lineHeight: 1.02
    letterSpacing: "-0.03em"
    fontVariation: "\"wdth\" 125"
  headline:
    fontFamily: "Dirstato Display, Dirstato Text, ui-monospace, Menlo, monospace"
    fontSize: "clamp(1.75rem, 3.8vw, 3.25rem)"
    fontWeight: 800
    lineHeight: 1.08
    letterSpacing: "-0.03em"
    fontVariation: "\"wdth\" 125"
  numeral:
    fontFamily: "Dirstato Display, Dirstato Text, ui-monospace, Menlo, monospace"
    fontSize: "clamp(2.5rem, 8vw, 5.5rem)"
    fontWeight: 800
    lineHeight: 1.05
    fontFeature: "\"tnum\" 1"
    fontVariation: "\"wdth\" 125"
  prompt:
    fontFamily: "Dirstato Text, ui-monospace, SF Mono, Menlo, monospace"
    fontSize: "18px"
    fontWeight: 600
    lineHeight: 1.5
  body:
    fontFamily: "Dirstato Text, ui-monospace, SF Mono, Menlo, monospace"
    fontSize: "16px"
    fontWeight: 400
    lineHeight: 1.5
    fontFeature: "\"calt\" 1"
  screen:
    fontFamily: "Dirstato Text, ui-monospace, SF Mono, Menlo, monospace"
    fontSize: "15px"
    fontWeight: 400
    lineHeight: 1.45
    fontFeature: "\"calt\" 0"
  label:
    fontFamily: "Dirstato Text, ui-monospace, SF Mono, Menlo, monospace"
    fontSize: "14px"
    fontWeight: 400
    lineHeight: 1.5
  caption:
    fontFamily: "Dirstato Text, ui-monospace, SF Mono, Menlo, monospace"
    fontSize: "13px"
    fontWeight: 400
    lineHeight: 1.5
rounded:
  none: "0px"
spacing:
  gutter: "clamp(16px, 4vw, 56px)"
  cell: "1ch"
  cell-2: "2ch"
  cell-3: "3ch"
  xs: "0.25rem"
  sm: "0.75rem"
  md: "1.5rem"
  lg: "2.5rem"
  section: "clamp(4.5rem, 12vh, 8rem)"
components:
  copy-button:
    backgroundColor: "{colors.ink}"
    textColor: "{colors.paper}"
    rounded: "{rounded.none}"
    padding: "0 1ch"
  copy-button-hover:
    backgroundColor: "transparent"
    textColor: "{colors.ink}"
  key-button:
    backgroundColor: "transparent"
    textColor: "{colors.ink}"
    typography: "{typography.label}"
    rounded: "{rounded.none}"
    padding: "0 1ch"
    height: "2.25rem"
    width: "4ch"
  key-button-hover:
    backgroundColor: "{colors.ink}"
    textColor: "{colors.paper}"
  link-hover:
    backgroundColor: "{colors.ink}"
    textColor: "{colors.paper}"
  screen:
    textColor: "{colors.ink}"
    typography: "{typography.screen}"
    rounded: "{rounded.none}"
    padding: "0.75em 1ch"
  screen-row-selected:
    backgroundColor: "{colors.ink}"
    textColor: "{colors.paper}"
  status-line:
    backgroundColor: "{colors.ink}"
    textColor: "{colors.paper}"
    typography: "{typography.label}"
    padding: "0.5rem clamp(16px, 4vw, 56px)"
  key-cap:
    textColor: "{colors.ink}"
    rounded: "{rounded.none}"
    padding: "0 0.5ch"
    width: "2.2ch"
---

# Design System: dirstato

## Overview

**Creative North Star: "One Character-Cell Surface"**

The site is a terminal profile, not a page about a terminal. Everything sits on one ground, on a monospace grid, and the world's devices are the TUI's own: box-drawing rules, block bars, ▸ ▾ markers, reverse-video selection, a `$` prompt with a block cursor. The page follows the system appearance: near-white paper with ink in light, ink-black ground with soft white in dark, faint ink for anything secondary. (The North Star is taken from the direction contract's THESIS and its form name, Text Mode. The user has not confirmed it as a name.)

Density is the TUI's: text at reading size, wide gutters, large gaps between sections, and display type set heavy and wide so the headlines read as type on a screen, not as a marketing hero. Colour is nearly absent. One coral accent exists and it belongs to the cursor. Depth does not exist: the surface is flat, divided only by rules and by reverse video.

Interaction is keyboard-first. The live demo is driven by dirstato's real keymap, hover answers with reverse video, and redraws are instant like a terminal's.

**Key Characteristics:**
- One flat ground per appearance; no cards, panels or raised surfaces.
- Every character on the monospace grid; horizontal spacing in `ch`.
- Box-drawing `─` rules and 1px hairlines are the only dividers.
- Reverse video (ink ground, paper text) for selection and hover.
- Coral only in the cursor family.
- Instant redraws; the only timed motion is the ticking scan replay, the demo's autoplay tour, the blinking cursor and smooth anchor scrolling.

## Colors

A monochrome terminal profile in two appearances, with one coral accent that is never used for anything but the cursor.

### Primary
- **Cursor Coral** (`coral`): the logo's top bar colour. Used for `::selection` (with the fixed `selection-text` ink on top, 6.48:1), the prompt's block cursor `█`, and the demo caret. It does not change with appearance. On the light paper it measures 2.72:1, so it marks the cursor position and never carries text or a state on its own.

### Secondary
- **Focus Ember** (`focus` / `focus-dark`): the 2px `:focus-visible` outline and the demo screen's focus border. A darker coral on light paper (4.46:1) and a lighter one on dark (7.57:1), so it stays visible where coral itself would not.

### Neutral
- **Paper** (`paper` / `paper-dark`): the one ground. Also the text colour inside reverse video.
- **Ink** (`ink` / `ink-dark`): all primary text, the reverse-video ground, the copy button's fill, the status line. 17.63:1 on light paper, 15.38:1 on dark.
- **Faint Ink** (`faint` / `faint-dark`): secondary text: the `$` prompt sign, the quiet half of the headline, captions, legend terms, figure labels, shell comments, the keypad's 1px stroke. 5.61:1 light, 6.29:1 dark.
- **Rule** (`rule` / `rule-dark`): box-drawing rules, 1px borders, link underlines, `kbd` strokes, the scrollbar thumb. Never text (1.59:1 light, 1.77:1 dark).

### Named Rules
**The Cursor Owns Coral Rule.** Coral marks where the cursor is: selection, the block cursor, the demo caret. Nothing else takes it, not links, not hover, not headings, not focus.

**The Reverse Video Rule.** Hover and selection answer by swapping ink and paper. A hovered link, a hovered key, a selected demo row and the status line all use ink ground with paper text.

**The Logo Is Not the Palette Rule.** The logo's amber, teal and blue bars and its dark gradient tile stay inside the logo. Only its coral enters the system, as the cursor.

## Typography

**Display Font:** Dirstato Display (falling back to Dirstato Text, then ui-monospace, Menlo)
**Body Font:** Dirstato Text (falling back to ui-monospace, SF Mono, Menlo)

Both are subsets of GitHub's Monaspace v1.400, renamed under the OFL, each a variable font with `wght` 200–800 and `wdth` 100–125 (read from the font files' `fvar` tables). The subsets do not name their source face. The direction contract names Xenon for display and Neon for text; the outlines agree (the display `n` has slab feet and twice the outline segments of the text `n`), but the mapping is unverified.

**Character:** a slab-footed monospace, set heavy and wide, over a plain grotesque monospace. At the default width both have the same advance (1240 of 2000 units); headlines run at width 125, so they are not on the body's grid.

### Hierarchy
- **Display** (`display`): the hero headline only, max 19ch. Its second line, the quiet claim, drops to weight 500 in faint ink.
- **Headline** (`headline`): section headings, max 22ch. The closing headline scales up to clamp(2rem, 5vw, 4.5rem), max 18ch.
- **Numeral** (`numeral`): the side-by-side figures that prove agreement with `du`, with tabular figures. On phones it becomes min(2.5rem, 10.5vw).
- **Prompt** (`prompt`): the install command after `$`. A second-tier prompt is 15px at weight 500; on phones both prompts are 15px.
- **Body** (`body`): running text, intros and rules, max 62–64ch. The lede is 17px (16px on phones), max 62ch.
- **Screen** (`screen`): the live demo; 12.5px on phones.
- **Label** (`label`): the top line, notes, transcripts, keypad, footer, status line.
- **Caption** (`caption`): the demo caption and the note's legend.

### Named Rules
**The One Grid Rule.** Every face is monospace, and every measure inside text (gaps between words and cells, indents, list column widths, text max widths) is given in `ch`, so text, rules and bars line up as cells. Page-level layout (gutter, caps, block columns) is the only exception.

**The No Ligatures in Cells Rule.** Body text keeps contextual alternates on (`calt` 1). The demo screen, transcripts, the race bars and the box-drawing rules turn them off (`calt` 0), so columns and rule characters never shift.

## Layout

One column inside a fluid gutter (`spacing.gutter`), with a sticky top line. Section rhythm is `spacing.section` above each block, a full-width box-drawing rule at the top of each block, and 2.5rem (`spacing.lg`) under that rule. Vertical spacing steps by `xs`, `sm`, `md` and `lg`; horizontal spacing is counted in characters (1ch, 2ch, 3ch). The hero and blocks cap at 88rem.

Breakpoints, all from `style.css`:
- **Below 760px (phone):** the screen drops to 12.5px, the top line keeps only `install` and `GitHub`, the race bars move to their own row, the numerals shrink.
- **760px and up:** definition lists become two columns (rules 32ch, keys 22ch, figures 18ch, with a 3ch gap).
- **Below 800px:** transcripts pre-wrap at 12.5px and show the 42-column render of the demo row (date and count dropped, path truncated with `…`) in place of the 82-column wide row.
- **1240px and up:** the demo session becomes two columns, the screen up to 95ch and a sticky note column of at least 28ch.
- **1360px and up:** each block becomes a sticky lead (5fr) beside a body (8fr), 4rem apart. The brief gives the reason for this threshold as the wide transcript row not fitting the body column below it; that is unverified here.

The demo screen measures its own width in characters and re-renders to fit (32 columns minimum). Like the real app it drops the date column below 90 columns and the count column below 70 (constants in `tui.js`, taken from the Go app). Anchored scrolling leaves 4.5rem for the sticky top line.

### Named Rules
**The Text Wraps, the Screen Re-Renders Rule.** Prose reflows; terminal output never reflows mid-row. It is re-rendered at a narrower column count, the way the real TUI does it.

## Elevation & Depth

None. The system is flat: one ground, no tonal layers, no blur, no glow, no elevation. Depth is replaced by three devices: rules, reverse video and the sticky top line, which paints the paper colour over content scrolling beneath it.

`box-shadow` appears only as a zero-blur 1px stroke, used where a border would change the box size:
- **Hairline stroke** (`inset 0 0 0 1px`): the copy button's hover and done states (ink), keypad keys (faint), `kbd` (rule).
- **Focus ring** (`0 0 0 1px` in focus colour): doubles the demo screen's 1px border to 2px when focused.

`text-shadow` appears once: block bars carry `0.06em 0 currentColor` to close the seams between `█` glyphs (checked in `.impeccable/review/zoom-bar-light.png`).

### Named Rules
**The Flat Surface Rule.** Nothing floats. A shadow with blur, an offset meant to suggest height, or a glow is outside this world.

## Shapes

Square everything (`rounded.none`). Buttons, keys, the demo screen, `kbd` and the status line have hard corners. Two kinds of rule divide the surface: box-drawing `─` characters for the full-width section dividers and the top line, and 1px CSS borders in rule colour for the demo screen and the note's legend. Bars are text (`█`, with `▌` for half cells), never drawn boxes. Markers are the TUI's text characters: ▸ ▾, `…`, `⏎`, `!`.

The one rounded, multi-colour shape on the page is the logo, a binding brand asset (PRODUCT.md), shown at 20px in the top line. It is exempt from this section.

## Components

### Buttons
Text-cell buttons: a word or a key, padded one character each side.
- **Shape:** square (`rounded.none`), height of one text line.
- **Copy (primary):** reverse video, ink fill with paper text, weight 500, 1ch from the command it copies.
- **Copy hover / done:** clears to the paper ground with ink text and an inset 1px ink stroke. The label reads "copied" for 2s, then returns to "copy".
- **Keypad key (secondary):** transparent with a faint 1px inset stroke, at least 4ch wide and 2.25rem tall. Hover, and the 260ms flash when the autoplay tour presses a key, fill it with reverse video.
- **Focus:** the global 2px focus outline, 3px offset.

### Links
- Inherit the text colour, with a 1px underline in rule colour at 0.25em offset.
- **Hover:** reverse video; the underline disappears. Navigation links in the top line have no underline at rest.

### Prompt
The install line, the page's primary action. Faint `$` (not selectable), the command in prompt type, the coral block cursor `█` after the command (hero and close only), then the copy button. Wrapped lines hang under the command by 2ch. Shell comments (`# …`) above a prompt are faint, 15px.

### Demo Screen (signature)
A live model of the dirstato TUI (`tui.js`, constants and legend from the Go app) inside a 1px rule border, padding 0.75em by 1ch. Rows are the app's own output: size, percentage, ten-cell block bar, count, date, tree markers. The selected row is reverse video; dimmed text is faint; bold is weight 700. The screen is focusable; on focus its border turns focus colour and the coral caret blinks. On wide screens a sticky note column explains the row under the cursor, with a legend in caption type below a 1px rule.

Motion is the terminal's: redraws are instant. The scan replay ticks every 100ms for 1200ms (a rescan for 700ms), then an autoplay tour presses a fixed script of keys every 1150ms until the script ends or the visitor touches anything. Replay starts when about 120px of the screen is in view (IntersectionObserver rootMargin -120px). Under reduced motion the screen draws its final state at once, with no scan or rescan replay, no tour and no blinking.

The demo must match the real binary: `scripts/compare-demo.py` checks it against the binary's output. Any change to the demo's rendering stays faithful to the app.

### Status Line
A full-width bar fixed to the bottom of the viewport in reverse video, label type, that shows a transient message (the copied command, or a fallback telling the visitor to press ⌘C) for 3s, then hides. It appears and disappears without a transition.

### Keys (`kbd`)
Inline key caps: weight 500, at least 2.2ch wide, 0.5ch side padding, square, an inset 1px rule-colour stroke.

## Do's and Don'ts

### Do:
- **Do** keep every surface on the single paper ground; divide it with box-drawing `─` rules or 1px rule-colour hairlines.
- **Do** answer hover and selection with reverse video (ink ground, paper text).
- **Do** keep coral to the cursor family: `::selection`, the block cursor, the demo caret.
- **Do** use the `focus` colour (2px outline, 3px offset) for keyboard focus.
- **Do** give horizontal measures in `ch` and keep all type monospace.
- **Do** wrap runs of `█` in the bar treatment (`0.06em 0 currentColor` text-shadow) so they read as one solid bar.
- **Do** turn off contextual alternates (`calt` 0) in any terminal output, bars and box-drawing rules.
- **Do** re-render terminal output at a narrower column count instead of letting it wrap mid-row; drop date, then count, like the app.
- **Do** keep redraws instant and skip the scan replay, tour, blink and smooth scroll under reduced motion.

### Don't:
- **Don't** use coral for links, hover, headings, focus or emphasis.
- **Don't** use blurred, glowing or height-suggesting shadows; `box-shadow` is only a zero-blur 1px stroke, and `text-shadow` only the bar seam.
- **Don't** round corners or put content in cards or panels; the logo is the only rounded shape.
- **Don't** bring the logo's amber, teal or blue into the interface.
- **Don't** set coral text or coral-only indicators on the light paper (2.72:1).
- **Don't** add colour to the demo; it stays monochrome like the real TUI.
