package ui

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"weak"

	tea "charm.land/bubbletea/v2"

	"github.com/lemiorhan/dirstato/internal/scan"
)

func write(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, bytes.Repeat([]byte{0xA5}, size), 0o644); err != nil {
		t.Fatal(err)
	}
}

func scanned(t *testing.T, root string) Model {
	t.Helper()
	return scannedWith(t, Config{Root: root})
}

func scannedWith(t *testing.T, cfg Config) Model {
	t.Helper()
	root := cfg.Root
	cfg.Copy = func(string) error { return nil }
	m := New(cfg)
	m, _ = m.update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m, _ = m.update(m.startCmd())
	if m.tree == nil {
		t.Fatalf("scan of %s failed: %v", root, m.err)
	}
	return m
}

func find(t *testing.T, m Model, rel string) *scan.Node {
	t.Helper()
	n := m.tree.Root
	for _, name := range strings.Split(rel, "/") {
		if n = childNamed(n, name); n == nil {
			t.Fatalf("no node at %s", rel)
		}
	}
	return n
}

func weakDescendants(n *scan.Node) []weak.Pointer[scan.Node] {
	var ws []weak.Pointer[scan.Node]
	var walk func(*scan.Node)
	walk = func(n *scan.Node) {
		for _, c := range n.Children {
			ws = append(ws, weak.Make(c))
			walk(c)
		}
	}
	walk(n)
	return ws
}

func selectedPath(m Model) string {
	r, ok := m.selected()
	if !ok {
		return ""
	}
	return r.target().Path()
}

// After a rescan the view must look as it did, and nothing in the Model may
// keep the replaced subtree alive.
func TestRescanKeepsViewAndReleasesOldNodes(t *testing.T) {
	for _, view := range []viewMode{treeView, drillView} {
		t.Run([]string{"tree", "drill-down"}[view], func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "root")
			write(t, filepath.Join(root, "top"), 1<<20)
			write(t, filepath.Join(root, "a/b/f1"), 64<<10)
			write(t, filepath.Join(root, "a/b/f2"), 32<<10)
			write(t, filepath.Join(root, "a/g"), 16<<10)
			m := scanned(t, root)

			m.expanded[find(t, m, "a")] = true
			m.expanded[find(t, m, "a/b")] = true
			m.view = view
			m.cwd = find(t, m, "a/b")
			m.rebuild()
			m.selectNode(find(t, m, "a/b/f1"))
			old := weakDescendants(find(t, m, "a"))
			if len(old) != 4 { // b, b/f1, b/f2, g
				t.Fatalf("%d old nodes to check, want 4", len(old))
			}

			// f0 sorts above f1, and g and f2 go away: the selected row moves,
			// and the tree view's last rows, all under a, get dropped.
			write(t, filepath.Join(root, "a/b/f0"), 128<<10)
			for _, gone := range []string{"a/g", "a/b/f2"} {
				if err := os.Remove(filepath.Join(root, gone)); err != nil {
					t.Fatal(err)
				}
			}
			m, _ = m.update(rescanDoneMsg{m.tree.ScanSubtree(context.Background(), find(t, m, "a"), scan.Options{}, nil)})

			if got, want := selectedPath(m), filepath.Join(root, "a/b/f1"); got != want {
				t.Errorf("selection after rescan = %q, want %q", got, want)
			}
			if !m.expanded[find(t, m, "a/b")] {
				t.Error("a/b collapsed after rescanning a")
			}
			if m.cwd != find(t, m, "a/b") {
				t.Errorf("cwd after rescan = %q, want the current a/b node", m.cwd.Path())
			}
			runtime.GC()
			for _, w := range old {
				if n := w.Value(); n != nil {
					t.Errorf("old node %q is still reachable after the rescan", n.Name)
				}
			}
			// Without it m is dead at the GC and every weak pointer clears, leak or not.
			runtime.KeepAlive(m)
		})
	}
}

// A rescanned folder that no longer exists falls back to its nearest surviving
// ancestor, and its subtree must still be freed although the folder's own node
// shares an allocation with live siblings.
// setup scans top, p/{keep, small, x/g, x/y/f} with p, p/x and p/x/y expanded
// and a threshold that hides p/small behind p's summary row.
func setup(t *testing.T) (string, Model) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "root")
	write(t, filepath.Join(root, "top"), 1<<20)
	write(t, filepath.Join(root, "p/keep"), 512<<10)
	write(t, filepath.Join(root, "p/small"), 10)
	write(t, filepath.Join(root, "p/x/g"), 64<<10)
	write(t, filepath.Join(root, "p/x/y/f"), 32<<10)
	m := scanned(t, root)
	m.min, m.minText = 8<<10, "8K"
	m.expanded[find(t, m, "p")] = true
	m.expanded[find(t, m, "p/x")] = true
	m.expanded[find(t, m, "p/x/y")] = true
	return root, m
}

func TestRescanOfGoneFolder(t *testing.T) {
	goneRescan := func(t *testing.T, root string, m Model) Model {
		t.Helper()
		x := find(t, m, "p/x")
		if err := os.RemoveAll(filepath.Join(root, "p/x")); err != nil {
			t.Fatal(err)
		}
		m, _ = m.update(rescanDoneMsg{m.tree.ScanSubtree(context.Background(), x, scan.Options{}, nil)})
		return m
	}

	t.Run("drill-down", func(t *testing.T) {
		root, m := setup(t)
		delete(m.expanded, find(t, m, "p")) // a gone entry must not open its ancestor
		m.view = drillView
		m.cwd = find(t, m, "p/x")
		m.rebuild()
		m.selectNode(find(t, m, "p/x/g"))
		old := weakDescendants(find(t, m, "p/x"))
		if len(old) != 3 { // g, y, y/f
			t.Fatalf("%d old nodes to check, want 3", len(old))
		}
		m = goneRescan(t, root, m)

		if m.cwd != find(t, m, "p") {
			t.Errorf("cwd after its folder is gone = %q, want p", m.cwd.Path())
		}
		if len(m.expanded) != 1 || !m.expanded[m.tree.Root] {
			t.Errorf("expanded after p/x is gone has %d entries, want only the root", len(m.expanded))
		}
		runtime.GC()
		for _, w := range old {
			if n := w.Value(); n != nil {
				t.Errorf("node %q under the gone folder is still reachable", n.Name)
			}
		}
		runtime.KeepAlive(m)
	})

	t.Run("tree selection", func(t *testing.T) {
		root, m := setup(t)
		m.rebuild()
		m.selectNode(find(t, m, "p/x/g"))
		m = goneRescan(t, root, m)
		// Compares the node, not the path: p's summary row also resolves to p.
		if r, ok := m.selected(); !ok || r.node != find(t, m, "p") {
			t.Errorf("selection after its folder is gone = %+v, want p's own row", r)
		}
	})

	t.Run("summary row", func(t *testing.T) {
		root, m := setup(t)
		m.rebuild()
		m.selectSummary(find(t, m, "p"))
		m = goneRescan(t, root, m)
		r, ok := m.selected()
		if !ok || r.node != nil || r.parent != find(t, m, "p") {
			t.Errorf("selection after rescan is not p's summary row: %+v", r)
		}
	})
}

func TestRescanOfFolderReplacedByFile(t *testing.T) {
	root, m := setup(t)
	m.view = drillView
	m.cwd = find(t, m, "p/x")
	m.rebuild()
	x := find(t, m, "p/x")
	if err := os.RemoveAll(filepath.Join(root, "p/x")); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(root, "p/x"), 10)
	m, _ = m.update(rescanDoneMsg{m.tree.ScanSubtree(context.Background(), x, scan.Options{}, nil)})

	if m.cwd != find(t, m, "p") {
		t.Errorf("cwd after p/x became a file = %q, want p", m.cwd.Path())
	}
	if m.expanded[find(t, m, "p/x")] {
		t.Error("p/x is still expanded after it became a file")
	}
}

func press(key string) tea.KeyPressMsg {
	switch key {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "right":
		return tea.KeyPressMsg{Code: tea.KeyRight}
	case "backspace":
		return tea.KeyPressMsg{Code: tea.KeyBackspace}
	case "left":
		return tea.KeyPressMsg{Code: tea.KeyLeft}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	}
	return tea.KeyPressMsg{Code: rune(key[0]), Text: key}
}

func enterThreshold(t *testing.T, m Model, text string) Model {
	t.Helper()
	m, _ = m.update(press("m"))
	for range len(m.input.Value()) {
		m, _ = m.update(press("backspace"))
	}
	for _, r := range text {
		m, _ = m.update(press(string(r)))
	}
	m, _ = m.update(press("enter"))
	if m.minText != text {
		t.Fatalf("threshold text after typing %q = %q", text, m.minText)
	}
	return m
}

func isSummaryOf(r row, parent *scan.Node) bool { return r.node == nil && r.parent == parent }

// emptyFolderBelowThreshold is drill-down inside the empty folder p/e with a
// threshold that hides p/e: nothing is selected, so no earlier selection can
// steer where the cursor goes next.
func emptyFolderBelowThreshold(t *testing.T) Model {
	t.Helper()
	root := filepath.Join(t.TempDir(), "root")
	write(t, filepath.Join(root, "top"), 1<<20)
	write(t, filepath.Join(root, "p/keep"), 512<<10)
	if err := os.Mkdir(filepath.Join(root, "p/e"), 0o755); err != nil {
		t.Fatal(err)
	}
	m := scanned(t, root)
	m.view = drillView
	m.cwd = find(t, m, "p/e")
	m.rebuild()
	m = enterThreshold(t, m, "8K")
	if _, ok := m.selected(); ok {
		t.Fatal("empty folder p/e has a selected row")
	}
	return m
}

// ←, Tab and a new threshold can leave the selected row with no row of its
// own; the selection then goes to the row standing for it, never to whatever
// row sits at the old index.
func TestSelectionFallsToNearestRow(t *testing.T) {
	t.Run("← out of a folder below the threshold", func(t *testing.T) {
		_, m := setup(t)
		m.view = drillView
		m.cwd = find(t, m, "p/x")
		m.rebuild()
		m = enterThreshold(t, m, "200K") // p/x is 96K
		m, _ = m.update(press("left"))
		if r, _ := m.selected(); !isSummaryOf(r, find(t, m, "p")) {
			t.Errorf("selection = %+v, want p's summary row, which hides p/x", r)
		}
	})

	t.Run("← out of an empty folder below the threshold", func(t *testing.T) {
		m := emptyFolderBelowThreshold(t)
		m, _ = m.update(press("left"))
		if r, _ := m.selected(); !isSummaryOf(r, find(t, m, "p")) {
			t.Errorf("selection = %+v, want p's summary row, which hides p/e", r)
		}
	})

	t.Run("Tab from the summary of a folder below the threshold", func(t *testing.T) {
		_, m := setup(t)
		m.view = drillView
		m.cwd = find(t, m, "p/x")
		m.rebuild()
		m = enterThreshold(t, m, "200K")
		if r, _ := m.selected(); !isSummaryOf(r, find(t, m, "p/x")) {
			t.Fatalf("selection in p/x = %+v, want its summary row", r)
		}
		m, _ = m.update(press("tab"))
		if r, _ := m.selected(); !isSummaryOf(r, find(t, m, "p")) {
			t.Errorf("selection = %+v, want p's summary row, which hides p/x", r)
		}
	})

	t.Run("Tab from an empty folder below the threshold", func(t *testing.T) {
		m := emptyFolderBelowThreshold(t)
		m, _ = m.update(press("tab"))
		if r, _ := m.selected(); !isSummaryOf(r, find(t, m, "p")) {
			t.Errorf("selection = %+v, want p's summary row, which hides p/e", r)
		}
	})

	t.Run("threshold hides the selected file", func(t *testing.T) {
		_, m := setup(t)
		m.rebuild()
		if !m.selectNode(find(t, m, "p/x/y/f")) {
			t.Fatal("p/x/y/f has no row")
		}
		// 40K hides y (32K) and f with it, but not g (64K): the old index now
		// holds p's summary row.
		m = enterThreshold(t, m, "40K")
		if r, _ := m.selected(); !isSummaryOf(r, find(t, m, "p/x")) {
			t.Errorf("selection = %+v, want p/x's summary row, which hides y", r)
		}
	})
}

func TestThresholdShowsWhatWasTyped(t *testing.T) {
	root := filepath.Join(t.TempDir(), "root")
	write(t, filepath.Join(root, "f"), 1<<20)
	m := scannedWith(t, Config{Root: root, Min: 3 << 29, MinText: "1.5G"})
	if h := strings.Join(m.header(), "\n"); !strings.Contains(h, "min 1.5G") {
		t.Errorf("header %q does not show the --min text", h)
	}
	m, _ = m.update(press("m"))
	if !m.editing {
		t.Fatal("m did not open the threshold input")
	}
	if got := m.input.Value(); got != "1.5G" {
		t.Errorf("m input prefilled with %q, want the --min text 1.5G", got)
	}

	// A formatted label would show 0.1G as 102.4M.
	for range len("1.5G") {
		m, _ = m.update(press("backspace"))
	}
	for _, r := range " 0.1G" {
		m, _ = m.update(press(string(r)))
	}
	m, _ = m.update(press("enter"))
	if m.min != 107374182 {
		t.Fatalf("threshold after typing 0.1G = %d, want 107374182", m.min)
	}
	if h := strings.Join(m.header(), "\n"); !strings.Contains(h, "min 0.1G ") {
		t.Errorf("header %q does not show the typed text 0.1G", h)
	}
	m, _ = m.update(press("m"))
	m, _ = m.update(press("enter"))
	if m.min != 107374182 {
		t.Errorf("threshold after m and enter = %d, want it unchanged", m.min)
	}
}

func TestRightOnSummaryRowGoesToFolderStart(t *testing.T) {
	for _, view := range []viewMode{treeView, drillView} {
		t.Run([]string{"tree", "drill-down"}[view], func(t *testing.T) {
			_, m := setup(t)
			m.view = view
			m.cwd = find(t, m, "p")
			m.rebuild()
			if !m.selectSummary(find(t, m, "p")) {
				t.Fatal("p has no summary row")
			}
			m, _ = m.update(press("right"))
			if r, _ := m.selected(); r.node != find(t, m, "p/keep") {
				t.Errorf("→ on p's summary row selected %+v, want p's first row p/keep", r)
			}
		})
	}
}

// When the row a selection stood for is gone after a rescan, the selection
// goes to the nearest row standing for the same path, never to whatever row
// now sits at the old index.
func TestRescanSelectionFallsToNearestRow(t *testing.T) {
	rescan := func(t *testing.T, m Model, n *scan.Node) Model {
		t.Helper()
		m, _ = m.update(rescanDoneMsg{m.tree.ScanSubtree(context.Background(), n, scan.Options{}, nil)})
		return m
	}

	for _, view := range []viewMode{treeView, drillView} {
		t.Run("summary of a folder that became a file/"+[]string{"tree", "drill-down"}[view], func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "root")
			write(t, filepath.Join(root, "top"), 1<<20)
			write(t, filepath.Join(root, "p/keep"), 512<<10)
			write(t, filepath.Join(root, "p/x/g"), 64<<10)
			write(t, filepath.Join(root, "p/x/s"), 10)
			write(t, filepath.Join(root, "p/z"), 16<<10)
			m := scanned(t, root)
			m.min, m.minText = 8<<10, "8K"
			m.expanded[find(t, m, "p")] = true
			m.expanded[find(t, m, "p/x")] = true
			m.view = view
			m.cwd = find(t, m, "p/x")
			m.rebuild()
			x := find(t, m, "p/x")
			if !m.selectSummary(x) {
				t.Fatal("p/x has no summary row")
			}
			if err := os.RemoveAll(filepath.Join(root, "p/x")); err != nil {
				t.Fatal(err)
			}
			// Larger than keep, so p/x's row is not at the old cursor index.
			write(t, filepath.Join(root, "p/x"), 600<<10)
			m = rescan(t, m, x)
			if r, _ := m.selected(); r.node != find(t, m, "p/x") {
				t.Errorf("selection = %+v, want the row of the file p/x", r)
			}
		})
	}

	t.Run("summary row that is gone", func(t *testing.T) {
		root, m := setup(t)
		m.rebuild()
		p := find(t, m, "p")
		if !m.selectSummary(p) {
			t.Fatal("p has no summary row")
		}
		if err := os.Remove(filepath.Join(root, "p/small")); err != nil {
			t.Fatal(err)
		}
		m = rescan(t, m, p)
		if r, _ := m.selected(); r.node != p {
			t.Errorf("selection = %+v, want p's own row", r)
		}
	})

	t.Run("summary row of a replaced folder that is gone", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "root")
		write(t, filepath.Join(root, "top"), 1<<20)
		write(t, filepath.Join(root, "p/x/g"), 64<<10)
		write(t, filepath.Join(root, "p/x/t"), 10)
		write(t, filepath.Join(root, "p/x/y/f"), 32<<10)
		write(t, filepath.Join(root, "p/x/y/s"), 10)
		m := scanned(t, root)
		m.min, m.minText = 8<<10, "8K"
		for _, d := range []string{"p", "p/x", "p/x/y"} {
			m.expanded[find(t, m, d)] = true
		}
		m.rebuild()
		if !m.selectSummary(find(t, m, "p/x/y")) {
			t.Fatal("p/x/y has no summary row")
		}
		if err := os.Remove(filepath.Join(root, "p/x/y/s")); err != nil {
			t.Fatal(err)
		}
		// The old y node still points at x, whose summary row (t) survives.
		m = rescan(t, m, find(t, m, "p/x"))
		if r, _ := m.selected(); r.node != find(t, m, "p/x/y") {
			t.Errorf("selection = %+v, want the new p/x/y row", r)
		}
	})

	t.Run("file now below the threshold", func(t *testing.T) {
		root, m := setup(t)
		m.rebuild()
		if !m.selectNode(find(t, m, "p/x/g")) {
			t.Fatal("p/x/g has no row")
		}
		write(t, filepath.Join(root, "p/x/g"), 10)
		m = rescan(t, m, find(t, m, "p/x"))
		if r, _ := m.selected(); r.node != nil || r.parent != find(t, m, "p/x") {
			t.Errorf("selection = %+v, want p/x's summary row, which now hides g", r)
		}
	})

	t.Run("drill-down folder that is gone", func(t *testing.T) {
		root, m := setup(t)
		m.view = drillView
		m.cwd = find(t, m, "p")
		m.rebuild()
		x := find(t, m, "p/x")
		if !m.selectNode(x) {
			t.Fatal("p/x has no row")
		}
		if err := os.RemoveAll(filepath.Join(root, "p/x")); err != nil {
			t.Fatal(err)
		}
		// Rows are now keep and p's summary; the old index 1 is the summary.
		m = rescan(t, m, x)
		if r, _ := m.selected(); r.node != find(t, m, "p/keep") {
			t.Errorf("selection = %+v, want the first row p/keep", r)
		}
	})
}

func TestHelpKeepsCloseHintOnShortTerminals(t *testing.T) {
	_, m := setup(t)
	m.help = true
	for _, h := range []int{24, 10} {
		m.height = h
		lines := strings.Split(m.render(), "\n")
		if len(lines) > h || lines[len(lines)-1] != helpClose {
			t.Errorf("height %d: help has %d lines ending %q, want at most %d ending with the close hint",
				h, len(lines), lines[len(lines)-1], h)
		}
	}
	m.height = 24
	if lines := strings.Split(m.render(), "\n"); len(lines) != 1+len(helpText) {
		t.Errorf("height 24: help shows %d lines, want all %d", len(lines), 1+len(helpText))
	}
}
