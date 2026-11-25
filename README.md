<p align="center">
  <img src="docs/assets/logo.svg" width="112" height="112" alt="dirstato logo">
</p>

<h1 align="center">dirstato</h1>

<p align="center">
  <strong>See what is eating your disk, right in the terminal.</strong><br>
  A fast, read-only disk usage explorer for macOS whose numbers match <code>du</code>.
</p>

<p align="center">
  <a href="LICENSE"><img alt="License: Apache 2.0" src="https://img.shields.io/badge/license-Apache%202.0-blue.svg"></a>
  <img alt="Go 1.27+" src="https://img.shields.io/badge/go-1.27%2B-00ADD8.svg?logo=go&amp;logoColor=white">
  <img alt="Platform: macOS" src="https://img.shields.io/badge/platform-macOS-lightgrey.svg?logo=apple">
</p>

---

dirstato scans a folder in parallel, then lets you walk the result as an
expandable tree or folder by folder. Every row shows how much disk it uses,
its share of its parent, how many files it holds and when something inside it
last changed. It never deletes or modifies anything.

```text
dirstato  /opt/homebrew/Cellar/go/1.27.4/libexec · size desc · min 4M · tree
────────────────────────────────────────────────────────────────────────────────────────────
 270.6M 100% ██████████    16K 2026-11-02 ▾ /opt/homebrew/Cellar/go/1.27.4/libexec
 156.1M  58% ██████        12K 2026-11-02   ▾ src/
  63.7M  41% ████         4.5K 2026-11-02     ▸ cmd/
  18.1M  12% █            1.2K 2026-11-02     ▸ runtime/
  17.2M  11% █            1.2K 2026-11-02     ▸ crypto/
  11.6M   7% █            1.4K 2026-11-02     ▸ internal/
   5.9M   4% █             346 2026-11-02     ▸ syscall/
   5.4M   3% █             461 2026-11-02     ▸ net/
   4.1M   3% █             559 2026-11-02     ▸ go/
  30.3M  19% ██           2.3K                  … 70 küçük öğe
  66.2M  24% ██             13 2026-11-02   ▸ pkg/
  19.0M   7% █            3.5K 2026-11-02   ▸ test/
  18.3M   7% █               2 2026-11-02   ▸ bin/
   8.6M   3% █              30 2026-11-02   ▸ api/
   2.3M   1% █              58                … 9 küçük öğe
────────────────────────────────────────────────────────────────────────────────────────────
↑↓ hareket · →← aç/kapat · Tab görünüm · s sırala · m min · c kopyala · r tara · ? yardım · q çık
```

<sub>dirstato scanning a Go installation with <code>--min 4M</code>. Items under
the threshold are folded into one summary row per folder (<em>“70 küçük öğe”</em>
means “70 small items”).</sub>

> [!NOTE]
> The interface text is currently in Turkish. Keys, columns and flags are
> documented in English below.

## Features

- **Numbers you can trust.** Sizes are allocated disk blocks, the same measure
  as `du -skx`. Hard links are counted once, symlinks are not followed and the
  scan stays on one file system. The test suite checks the root total against
  `/usr/bin/du` on a fixture tree.
- **Fast.** Folders are scanned in parallel; a progress screen shows files,
  folders, bytes and the current path while it runs.
- **Two views.** An expandable tree and a drill-down view that shows one
  folder at a time. `Tab` switches between them and keeps your selection.
- **Five columns.** Size, percentage of the parent, a bar, the recursive file
  count and the newest modification date in the subtree.
- **Sort and filter.** Cycle through size descending, size ascending and name.
  Hide everything below a size with `--min` or the `m` key; hidden items are
  summed into one row per folder.
- **Rescan in place.** `r` rescans the selected folder after you clean up and
  updates every parent total.
- **Copy the path.** `c` copies the absolute path to the clipboard, quoted for
  a POSIX shell when needed.
- **Read-only.** There is no delete key. dirstato tells you where the space
  went; you decide what to do about it.

## Installation

dirstato runs on **macOS** only: it reads file system metadata through macOS
system calls and copies with `pbcopy`. Building it requires
[Go](https://go.dev/dl/) 1.27 or newer.

```sh
git clone https://github.com/lemiorhan/dirstato.git
cd dirstato
go build -o dirstato .
mv dirstato /usr/local/bin/   # or anywhere on your PATH
```

To try it without installing, `./start.sh [path] [--min SIZE]` builds into a
temporary folder and runs it from your current directory.

A Homebrew tap is planned.

## Usage

```text
dirstato [path] [--min SIZE]
```

| Argument      | Meaning                                                                    |
| ------------- | -------------------------------------------------------------------------- |
| `path`        | Folder to scan. Defaults to the current directory.                         |
| `--min SIZE`  | Hide items smaller than `SIZE`. Also accepted as `--min=SIZE`.             |
| `-h`, `--help`| Print usage and exit.                                                      |

Flags may come before or after the path. `SIZE` uses base 1024 and accepts an
optional unit, case-insensitive, with an optional `B` or `iB` suffix:
`500M`, `1.5G`, `100K`, `2T`, `512KiB`. A bare number is bytes; `0` turns the
filter off.

```sh
dirstato                     # scan the current folder
dirstato ~/Library --min 1G  # only show items of 1 GiB or more
dirstato /Applications
```

If the path is a symlink to a folder, its target is scanned.

### Keys

| Key                   | Action                                                                     |
| --------------------- | -------------------------------------------------------------------------- |
| `↑` `↓` / `k` `j`     | Move up / down one row                                                     |
| `PgUp` / `PgDn`       | Move up / down one page                                                    |
| `Home` `g` / `End` `G`| Jump to the first / last row                                               |
| `→` / `l` / `Enter`   | Tree: expand the folder, or step into its first child. Drill-down: enter   |
| `←` / `h`             | Tree: collapse the folder, or jump to its parent. Drill-down: go up        |
| `Tab`                 | Switch between tree and drill-down, keeping the selection                  |
| `s`                   | Cycle sorting: size descending → size ascending → name                     |
| `m`                   | Set the minimum size (`Enter` applies, `Esc` cancels, `0` turns it off)    |
| `c`                   | Copy the selected item's absolute path                                     |
| `r`                   | Rescan the selected folder; on a file, rescan its parent folder            |
| `?`                   | Show or hide the help screen                                               |
| `q` / `Ctrl+C`        | Quit; during a scan, cancel it and quit                                    |

On a summary row, `→`, `c` and `r` act on the folder that holds the hidden
items.

### Columns

| Column        | Meaning                                                                  |
| ------------- | ------------------------------------------------------------------------ |
| size          | Disk usage in base 1024, the same measure as `du -skx`                   |
| %, bar        | Share of the parent folder                                               |
| count         | Number of files in the subtree (folders are not counted)                 |
| date          | Newest file modification time in the subtree, as `YYYY-MM-DD`            |
| `!`           | The folder could not be read; its size may be incomplete                 |

On narrow terminals the date column is dropped first, then the count. The
widths are the `dropDateBelow` and `dropCountBelow` constants in
[`internal/ui/render.go`](internal/ui/render.go).

The header shows the root, sort order, size threshold, current view and the
number of folders that could not be read. Dotfiles are always shown.

## How sizes are counted

dirstato aims to agree with `du -skx` byte for byte:

- **Disk usage, not file length.** Each entry counts `st_blocks × 512`, so a
  sparse file counts what it actually occupies. A folder's own blocks count too.
- **Hard links count once.** When several paths share an inode, the path that
  sorts first lexicographically carries the bytes and the others show `0`.
- **Symlinks are not followed.** A symlink counts as itself. Only the root is
  resolved: when you pass a symlink to a folder, its target is scanned, which
  corresponds to `du -skx path/` with a trailing slash.
- **One file system.** The scan does not descend into folders on another
  device, such as mounted volumes.
- **Unreadable folders are marked, not fatal.** They get a `!`, keep their own
  blocks and the scan continues.

These rules are pinned by tests that compare dirstato with `/usr/bin/du` on
fixture trees containing nested folders, hard links, symlinks, sparse files,
unreadable folders and a mounted disk image: see
[`internal/scan/scan_eval_test.go`](internal/scan/scan_eval_test.go)
(`TestEvalScanMatchesDu`, `TestEvalRescanMatchesDu`, `TestEvalMountBoundary`).

## Performance

All nodes are kept in memory so that the views, sorting and filtering are
instant after the scan. One measurement, on the author's machine on
2026-11-24, scanning a `~/Code` folder:

| Measure                          | Result                     |
| -------------------------------- | -------------------------- |
| dirstato scan                    | 11.6 s                     |
| `du -skx`, same command          | 46.5 s                     |
| Nodes                            | 1.57 million               |
| Heap after GC                    | 183 MB (116 bytes per node)|
| Peak RSS                         | 288 MB                     |
| Root size vs `du`                | identical                  |

To measure a folder of your own:

```sh
DIRSTATO_EVAL_DIR=~/Code go test -run RealDir -v ./internal/scan
```

## Development

```sh
go build ./...
go vet ./...
go test ./...
```

The tests run on macOS and need `/usr/bin/du` and `hdiutil`, both part of the
system. `TestEvalRealDir` only runs when `DIRSTATO_EVAL_DIR` is set.

```text
main.go             command line parsing and program start
internal/scan       parallel scanner, hard link resolution, subtree rescans
internal/ui         Bubble Tea model, views and rendering
internal/units      size parsing and formatting (base 1024)
internal/clip       shell quoting and clipboard copy
docs/               design decisions (spec.md) and plan (plan.md), in Turkish
```

dirstato is built with [Bubble Tea](https://github.com/charmbracelet/bubbletea),
[Bubbles](https://github.com/charmbracelet/bubbles) and
[Lip Gloss](https://github.com/charmbracelet/lipgloss).

## Contributing

Bug reports, ideas and pull requests are welcome. Please read
[CONTRIBUTING.md](CONTRIBUTING.md) first. Everyone taking part is expected to
follow the [Code of Conduct](CODE_OF_CONDUCT.md). For security issues, see
[SECURITY.md](SECURITY.md).

## License

dirstato is licensed under the [Apache License 2.0](LICENSE).
Copyright 2026 Lemi Orhan Ergin.
