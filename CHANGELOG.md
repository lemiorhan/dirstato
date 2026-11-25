# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-11-25

First release. Install with `brew install lemiorhan/dirstato/dirstato`.

### Added

- Parallel, read-only disk usage scan whose root total matches `du -skx`:
  allocated blocks, hard links counted once, no symlink following, no mount
  crossing.
- Progress screen during the scan; `q` or `Ctrl+C` cancels it.
- Tree and drill-down views, switched with `Tab` while keeping the selection.
- Size, percentage, bar, recursive file count and newest modification date
  columns; date and count columns drop on narrow terminals.
- Sorting by size descending, size ascending and name (`s`).
- Minimum size filter with `--min` and `m`, with a summary row per folder for
  hidden items.
- Copying the selected absolute path, shell-quoted when needed (`c`).
- Rescanning the selected folder in place (`r`).
- Help screen (`?`).
- English user interface and command line messages.

[Unreleased]: https://github.com/lemiorhan/dirstato/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/lemiorhan/dirstato/releases/tag/v0.1.0
