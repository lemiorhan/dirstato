// Command scan-data captures the landing page demo's data: it scans a folder
// with dirstato's own scanner and writes the result as site/scan-data.js.
//
//	go run ./scripts/scan-data /opt/homebrew/Cellar/go/1.27.4/libexec site/scan-data.js
//
// The page shows the folder's size in bytes and as du -skx's KiB, so run
// compares the scan against du before it writes. site/index.html repeats some
// of the numbers in its static screen and du block; those are updated by hand.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/lemiorhan/dirstato/internal/scan"
)

// The demo folds smaller items into one summary row per folder, as
// dirstato --min does, which keeps the file small; tui.js reads it as meta.floor.
const floor = 128 << 10

type meta struct {
	Path    string  `json:"path"`
	DuKiB   int64   `json:"duKiB"`
	Bytes   int64   `json:"bytes"`
	Folders int     `json:"folders"`
	Floor   int64   `json:"floor"`
	Scanned string  `json:"scanned"`
	Seconds float64 `json:"seconds"`
}

type node struct {
	Name     string  `json:"n"`
	Size     int64   `json:"s"`
	Files    uint32  `json:"f"`
	Newest   string  `json:"d,omitempty"`
	Children []*node `json:"c,omitempty"`
	Folded   []int64 `json:"r,omitempty"` // count, size, files of the items under floor
	Dir      bool    `json:"x,omitempty"`
}

func convert(n *scan.Node) *node {
	o := &node{Name: n.Name, Size: n.Size, Files: n.Files, Newest: n.Newest().Format("2006-01-02")}
	if !n.IsDir() {
		return o
	}
	o.Dir = true
	var count, size, files int64
	for _, c := range n.Children {
		if c.Size < floor {
			count++
			size += c.Size
			files += int64(c.Files)
			continue
		}
		o.Children = append(o.Children, convert(c))
	}
	if count > 0 {
		o.Folded = []int64{count, size, files}
	}
	return o
}

func folders(n *scan.Node) int {
	if !n.IsDir() {
		return 0
	}
	total := 1
	for _, c := range n.Children {
		total += folders(c)
	}
	return total
}

func duKiB(path string) (int64, error) {
	out, err := exec.Command("/usr/bin/du", "-skx", path).Output()
	if err != nil {
		return 0, fmt.Errorf("du -skx %s: %w", path, err)
	}
	field, _, _ := strings.Cut(string(out), "\t")
	return strconv.ParseInt(field, 10, 64)
}

func run(folder, out string) error {
	start := time.Now()
	t, err := scan.Scan(context.Background(), folder, scan.Options{}, &scan.Progress{})
	if err != nil {
		return err
	}
	took := time.Since(start)
	if t.Errors() > 0 {
		return fmt.Errorf("%s: %d items could not be read", folder, t.Errors())
	}
	du, err := duKiB(t.Root.Name)
	if err != nil {
		return err
	}
	if t.Root.Size != du*1024 {
		return fmt.Errorf("dirstato counts %d bytes and du -skx reports %d KiB; the page needs the bytes to be exactly that many KiB", t.Root.Size, du)
	}

	m := meta{
		Path:    t.Root.Name,
		DuKiB:   du,
		Bytes:   t.Root.Size,
		Folders: folders(t.Root),
		Floor:   floor,
		Scanned: start.Format("2006-01-02"),
		Seconds: math.Round(took.Seconds()*100) / 100,
	}
	var buf bytes.Buffer
	buf.WriteString("globalThis.DIRSTATO_SCAN = ")
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(struct {
		Meta meta  `json:"meta"`
		Root *node `json:"root"`
	}{m, convert(t.Root)}); err != nil {
		return err
	}
	buf.Truncate(buf.Len() - 1)
	buf.WriteString(";\n")

	tmp, err := os.CreateTemp(filepath.Dir(out), ".scan-data-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(buf.Bytes()); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), out); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "wrote %s: %d KiB, matching du -skx; %d folders in %.2f s\nsite/index.html repeats some of these numbers; update them by hand\n", out, du, m.Folders, m.Seconds)
	return nil
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: go run ./scripts/scan-data <folder> <output.js>")
		os.Exit(2)
	}
	if err := run(os.Args[1], os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, "scan-data:", err)
		os.Exit(1)
	}
}
