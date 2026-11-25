package scan

import (
	"bytes"
	"context"
	"errors"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
	"weak"

	"golang.org/x/sys/unix"
)

// duBytes is the oracle: macOS du, in 512-byte blocks so no rounding is
// involved. du exits 1 on an unreadable folder but still prints the total.
func duBytes(t *testing.T, path string) int64 {
	t.Helper()
	cmd := exec.Command("/usr/bin/du", "-sx", path)
	cmd.Env = append(os.Environ(), "BLOCKSIZE=512")
	out, err := cmd.Output()
	var exitErr *exec.ExitError
	if err != nil && !(errors.As(err, &exitErr) && len(out) > 0) {
		t.Fatalf("du %s: %v", path, err)
	}
	blocks, err := strconv.ParseInt(strings.Fields(string(out))[0], 10, 64)
	if err != nil {
		t.Fatalf("du output %q: %v", out, err)
	}
	return blocks * 512
}

func write(t *testing.T, path string, size int, mtime time.Time) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, bytes.Repeat([]byte{0xA5}, size), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatal(err)
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func child(t *testing.T, n *Node, names ...string) *Node {
	t.Helper()
	for _, name := range names {
		var found *Node
		for _, c := range n.Children {
			if c.Name == name {
				found = c
			}
		}
		if found == nil {
			t.Fatalf("%s has no child %q", n.Path(), name)
		}
		n = found
	}
	return n
}

var (
	day1 = time.Date(2024, 1, 10, 12, 0, 0, 0, time.UTC)
	day2 = time.Date(2025, 6, 20, 12, 0, 0, 0, time.UTC)
	day3 = time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC)
	day0 = time.Date(2023, 2, 1, 12, 0, 0, 0, time.UTC)
)

// fixture builds:
//
//	big            1 MiB
//	hl1            hard link to a/hl2 (other folder)
//	sym            symlink to big
//	sparse         50 MiB apparent, 1 byte written
//	empty/         no entries, mtime day0
//	locked/x       folder mode 000
//	noexec/f1, noexec/sub/f2  folder mode 0444: listable, entries cannot be stat'ed
//	a/hl2          200 KiB
//	a/b/x, a/b/y   hard links to one 300 KiB inode (same folder)
//	a/b/c/small    100 bytes, newest file (day3)
func fixture(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "root")
	write(t, filepath.Join(root, "big"), 1<<20, day1)
	write(t, filepath.Join(root, "a/hl2"), 200<<10, day2)
	must(t, os.Link(filepath.Join(root, "a/hl2"), filepath.Join(root, "hl1")))
	write(t, filepath.Join(root, "a/b/x"), 300<<10, day1)
	must(t, os.Link(filepath.Join(root, "a/b/x"), filepath.Join(root, "a/b/y")))
	write(t, filepath.Join(root, "a/b/c/small"), 100, day3)
	must(t, os.Symlink("big", filepath.Join(root, "sym")))
	ts := unix.NsecToTimespec(day1.UnixNano())
	must(t, unix.UtimesNanoAt(unix.AT_FDCWD, filepath.Join(root, "sym"), []unix.Timespec{ts, ts}, unix.AT_SYMLINK_NOFOLLOW))
	f, err := os.Create(filepath.Join(root, "sparse"))
	must(t, err)
	_, err = f.WriteAt([]byte{1}, 50<<20)
	must(t, err)
	must(t, f.Close())
	must(t, os.Chtimes(filepath.Join(root, "sparse"), day1, day1))
	must(t, os.Mkdir(filepath.Join(root, "empty"), 0o755))
	must(t, os.Chtimes(filepath.Join(root, "empty"), day0, day0))
	write(t, filepath.Join(root, "locked/x"), 10, day1)
	locked := filepath.Join(root, "locked")
	must(t, os.Chmod(locked, 0))
	t.Cleanup(func() { os.Chmod(locked, 0o755) })
	write(t, filepath.Join(root, "noexec/f1"), 10, day1)
	write(t, filepath.Join(root, "noexec/sub/f2"), 10, day1)
	noexec := filepath.Join(root, "noexec")
	must(t, os.Chmod(noexec, 0o444))
	t.Cleanup(func() { os.Chmod(noexec, 0o755) })
	return root
}

func scanOrFail(t *testing.T, root string) *Tree {
	t.Helper()
	tree, err := Scan(context.Background(), root, Options{Workers: 4}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return tree
}

func TestEvalScanMatchesDu(t *testing.T) {
	root := fixture(t)
	tree := scanOrFail(t, root)

	if got, want := tree.Root.Size, duBytes(t, root); got != want {
		t.Errorf("root size = %d, du = %d", got, want)
	}
	if got := tree.Errors(); got != 2 {
		t.Errorf("Errors() = %d, want 2 (locked, noexec)", got)
	}
	if !child(t, tree.Root, "locked").Unreadable() {
		t.Error("locked folder is not marked unreadable")
	}
	noexec := child(t, tree.Root, "noexec")
	if !noexec.Unreadable() {
		t.Error("noexec folder is not marked unreadable")
	}
	if len(noexec.Children) != 0 || noexec.Files != 0 {
		t.Errorf("noexec has %d children and %d files, want none: its entries cannot be stat'ed", len(noexec.Children), noexec.Files)
	}
	if child(t, tree.Root, "a").Unreadable() {
		t.Error("readable folder a is marked unreadable")
	}
	for _, sub := range [][]string{{"a"}, {"a", "b"}, {"a", "b", "c"}} {
		n := child(t, tree.Root, sub...)
		if got, want := n.Size, duBytes(t, n.Path()); got != want {
			t.Errorf("%s size = %d, du = %d", n.Path(), got, want)
		}
	}
	// big, hl1, sym, sparse, a/hl2, a/b/x, a/b/y, a/b/c/small; locked and noexec contribute none.
	if got := tree.Root.Files; got != 8 {
		t.Errorf("root Files = %d, want 8", got)
	}
	if got := child(t, tree.Root, "a", "b").Files; got != 3 {
		t.Errorf("a/b Files = %d, want 3", got)
	}
	if got := tree.Root.Newest(); !got.Equal(day3) {
		t.Errorf("root Newest = %v, want %v", got, day3)
	}
	if got := child(t, tree.Root, "a", "b").Newest(); !got.Equal(day3) {
		t.Errorf("a/b Newest = %v, want %v", got, day3)
	}
	if got := child(t, tree.Root, "empty").Newest(); !got.Equal(day0) {
		t.Errorf("empty Newest = %v, want its own mtime %v", got, day0)
	}
	// a/hl2 sorts before hl1, so a/hl2 carries the inode.
	if got := child(t, tree.Root, "hl1").Size; got != 0 {
		t.Errorf("hl1 size = %d, want 0 (a/hl2 owns the inode)", got)
	}
	if got := child(t, tree.Root, "a", "hl2").Size; got < 200<<10 {
		t.Errorf("a/hl2 size = %d, want at least %d", got, 200<<10)
	}
	if got := child(t, tree.Root, "sparse").Size; got >= 1<<20 {
		t.Errorf("sparse size = %d, want its allocated blocks, not 50 MiB", got)
	}
}

func rescan(t *testing.T, tree *Tree, n *Node) {
	t.Helper()
	res := tree.ScanSubtree(context.Background(), n, Options{Workers: 4}, nil)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	tree.Apply(res)
}

func TestEvalRescanMatchesDu(t *testing.T) {
	root := fixture(t)
	tree := scanOrFail(t, root)
	rootNode := tree.Root

	must(t, os.Remove(filepath.Join(root, "big")))
	sizeBefore := tree.Root.Size
	res := tree.ScanSubtree(context.Background(), tree.Root, Options{Workers: 4}, nil)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	child(t, tree.Root, "big")
	if tree.Root.Size != sizeBefore {
		t.Errorf("ScanSubtree changed the tree before Apply: root size %d -> %d", sizeBefore, tree.Root.Size)
	}
	tree.Apply(res)
	if tree.Root != rootNode {
		t.Error("Apply replaced the root node instead of updating it")
	}
	if got, want := tree.Root.Size, duBytes(t, root); got != want {
		t.Errorf("after deleting big: root size = %d, du = %d", got, want)
	}
	if got := tree.Errors(); got != 2 {
		t.Errorf("Errors() after rescanning the root = %d, want 2", got)
	}

	hl1 := child(t, tree.Root, "hl1")
	a := child(t, tree.Root, "a")
	must(t, os.Remove(filepath.Join(root, "a/hl2")))
	rescan(t, tree, a)
	if child(t, tree.Root, "a") != a {
		t.Error("Apply replaced folder a instead of updating it")
	}
	if got, want := tree.Root.Size, duBytes(t, root); got != want {
		t.Errorf("after deleting the owning link: root size = %d, du = %d", got, want)
	}
	if hl1.Size < 200<<10 {
		t.Errorf("hl1 size = %d, want the inode's blocks after a/hl2 is gone", hl1.Size)
	}
	if got, want := child(t, tree.Root, "a", "b", "c").Path(), filepath.Join(root, "a/b/c"); got != want {
		t.Errorf("path after rescanning a = %q, want %q", got, want)
	}

	c := child(t, tree.Root, "a", "b", "c")
	must(t, os.RemoveAll(filepath.Join(root, "a/b/c")))
	rescan(t, tree, c)
	for _, n := range child(t, tree.Root, "a", "b").Children {
		if n == c {
			t.Error("deleted folder a/b/c is still in the tree")
		}
	}
	if got, want := tree.Root.Size, duBytes(t, root); got != want {
		t.Errorf("after deleting a/b/c: root size = %d, du = %d", got, want)
	}
	if got := child(t, tree.Root, "a", "b").Newest(); !got.Equal(day1) {
		t.Errorf("a/b Newest = %v, want %v after its newest file is gone", got, day1)
	}

	must(t, os.Chmod(filepath.Join(root, "locked"), 0o755))
	locked := child(t, tree.Root, "locked")
	rescan(t, tree, locked)
	if locked.Unreadable() {
		t.Error("locked is still marked unreadable after it became readable")
	}
	if got := tree.Errors(); got != 1 {
		t.Errorf("Errors() = %d, want 1 once locked is readable", got)
	}
	if got, want := tree.Root.Size, duBytes(t, root); got != want {
		t.Errorf("after unlocking: root size = %d, du = %d", got, want)
	}
}

func weakDescendants(n *Node) []weak.Pointer[Node] {
	var ws []weak.Pointer[Node]
	var walk func(*Node)
	walk = func(n *Node) {
		for _, c := range n.Children {
			ws = append(ws, weak.Make(c))
			walk(c)
		}
	}
	walk(n)
	return ws
}

// assertReleased keeps tree alive through the GC: once the caller stops using
// it, the whole tree is garbage and every weak pointer clears, leak or not.
func assertReleased(t *testing.T, what string, ws []weak.Pointer[Node], want int, tree *Tree) {
	t.Helper()
	if len(ws) != want {
		t.Fatalf("%s: %d old nodes to check, want %d", what, len(ws), want)
	}
	runtime.GC()
	for _, w := range ws {
		if n := w.Value(); n != nil {
			t.Errorf("%s: old node %q is still reachable", what, n.Name)
		}
	}
	runtime.KeepAlive(tree)
}

// moveLinkLast puts the node at names last in its inode's link list, the
// position a filter reusing the backing array leaves behind in spare capacity.
func moveLinkLast(t *testing.T, tree *Tree, names ...string) {
	t.Helper()
	n := child(t, tree.Root, names...)
	k := tree.linkKey[n]
	tree.links[k] = append(slices.DeleteFunc(tree.links[k], func(o *Node) bool { return o == n }), n)
}

// TestEvalRescanReleasesOldNodes guards the memory budget across rescans: the
// replaced subtree must become garbage, so repeated rescans do not stack copies.
func TestEvalRescanReleasesOldNodes(t *testing.T) {
	root := fixture(t)
	write(t, filepath.Join(root, "solo/only/f"), 10, day1)
	write(t, filepath.Join(root, "pair/keep"), 10, day1)
	write(t, filepath.Join(root, "pair/gone/g"), 10, day1)
	write(t, filepath.Join(root, "pair/gone/deep/f"), 10, day1)
	tree := scanOrFail(t, root)

	moveLinkLast(t, tree, "a", "hl2")
	must(t, os.Remove(filepath.Join(root, "a/hl2")))
	// a/hl2, a/b, a/b/x, a/b/y, a/b/c, a/b/c/small
	old := weakDescendants(child(t, tree.Root, "a"))
	rescan(t, tree, child(t, tree.Root, "a"))
	assertReleased(t, "after rescanning a", old, 6, tree)

	// only, only/f
	old = weakDescendants(child(t, tree.Root, "solo"))
	must(t, os.RemoveAll(filepath.Join(root, "solo/only")))
	rescan(t, tree, child(t, tree.Root, "solo", "only"))
	assertReleased(t, "after solo/only is gone", old, 2, tree)

	// gone stays allocated with keep, so only what is below it can be freed:
	// g, deep, deep/f.
	old = weakDescendants(child(t, tree.Root, "pair", "gone"))
	must(t, os.RemoveAll(filepath.Join(root, "pair/gone")))
	rescan(t, tree, child(t, tree.Root, "pair", "gone"))
	assertReleased(t, "after pair/gone is gone", old, 3, tree)
}

func hdiutil(t *testing.T, args ...string) {
	t.Helper()
	if out, err := exec.Command("hdiutil", args...).CombinedOutput(); err != nil {
		t.Fatalf("hdiutil %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func TestEvalMountBoundary(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, "root")
	mnt := filepath.Join(root, "mnt")
	must(t, os.MkdirAll(mnt, 0o755))
	write(t, filepath.Join(root, "here"), 100<<10, day1)
	img := filepath.Join(tmp, "vol.dmg")
	hdiutil(t, "create", "-quiet", "-size", "8m", "-fs", "HFS+", "-volname", "dirstato-eval", img)
	hdiutil(t, "attach", "-quiet", "-nobrowse", "-noautoopen", "-noverify", "-mountpoint", mnt, img)
	t.Cleanup(func() { exec.Command("hdiutil", "detach", "-force", mnt).Run() })
	write(t, filepath.Join(mnt, "there"), 512<<10, day1)

	tree := scanOrFail(t, root)
	m := child(t, tree.Root, "mnt")
	if len(m.Children) != 0 {
		t.Errorf("scan crossed into the mounted volume: mnt has %d children", len(m.Children))
	}
	if got, want := tree.Root.Size, duBytes(t, root); got != want {
		t.Errorf("root size = %d, du -x = %d", got, want)
	}

	rescan(t, tree, m)
	if len(m.Children) != 0 {
		t.Errorf("rescan crossed into the mounted volume: mnt has %d children", len(m.Children))
	}
	if got, want := tree.Root.Size, duBytes(t, root); got != want {
		t.Errorf("after rescanning mnt: root size = %d, du -x = %d", got, want)
	}
}

const (
	fdLimit      = 64
	fdLimitDepth = 2 * fdLimit
)

func TestEvalDeepTreeStaysUnderFdLimit(t *testing.T) {
	root := filepath.Join(t.TempDir(), "root")
	deep := root
	for range fdLimitDepth {
		deep = filepath.Join(deep, "d")
	}
	write(t, filepath.Join(deep, "f"), 10, day1)

	var orig syscall.Rlimit
	must(t, syscall.Getrlimit(syscall.RLIMIT_NOFILE, &orig))
	low := orig
	low.Cur = fdLimit
	must(t, syscall.Setrlimit(syscall.RLIMIT_NOFILE, &low))
	t.Cleanup(func() { syscall.Setrlimit(syscall.RLIMIT_NOFILE, &orig) })

	// One worker never spawns, so every folder is walked inline on one stack.
	tree, err := Scan(context.Background(), root, Options{Workers: 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if tree.Errors() != 0 || tree.Root.Files != 1 {
		t.Errorf("%d folders deep under a %d fd limit: Errors() = %d, Files = %d; want 0 and 1",
			fdLimitDepth, fdLimit, tree.Errors(), tree.Root.Files)
	}
}

const realDirTolerance = 0.01

// TestEvalRealDir is the measurement gate: DIRSTATO_EVAL_DIR=~/Code go test -run RealDir -v ./internal/scan
func TestEvalRealDir(t *testing.T) {
	dir := os.Getenv("DIRSTATO_EVAL_DIR")
	if dir == "" {
		t.Skip("DIRSTATO_EVAL_DIR not set")
	}
	workers, _ := strconv.Atoi(os.Getenv("DIRSTATO_WORKERS"))
	if workers <= 0 {
		workers = defaultWorkers()
	}
	p := &Progress{}
	start := time.Now()
	tree, err := Scan(context.Background(), dir, Options{Workers: workers}, p)
	if err != nil {
		t.Fatal(err)
	}
	scanTime := time.Since(start)
	runtime.GC()
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	var ru syscall.Rusage
	must(t, syscall.Getrusage(syscall.RUSAGE_SELF, &ru))

	var nodes, dirs int
	var count func(*Node)
	count = func(n *Node) {
		nodes++
		if n.IsDir() {
			dirs++
		}
		for _, c := range n.Children {
			count(c)
		}
	}
	count(tree.Root)

	start = time.Now()
	du := duBytes(t, tree.Root.Path())
	duTime := time.Since(start)
	runtime.KeepAlive(tree)

	perNode := float64(ms.HeapAlloc) / float64(nodes)
	t.Logf("files=%d dirs=%d nodes=%d errors=%d", tree.Root.Files, dirs-1, nodes, tree.Errors())
	t.Logf("scan=%v du=%v workers=%d", scanTime.Round(time.Millisecond), duTime.Round(time.Millisecond), workers)
	t.Logf("heap after GC=%.0f MB, peak RSS (ru_maxrss)=%d, bytes/node=%.0f, nodes in 1 GB heap=%.1fM",
		float64(ms.HeapAlloc)/1e6, ru.Maxrss, perNode, 1e9/perNode/1e6)
	t.Logf("size=%d du=%d diff=%.4f%%", tree.Root.Size, du, 100*float64(tree.Root.Size-du)/float64(du))
	if diff := math.Abs(float64(tree.Root.Size-du)) / float64(du); diff > realDirTolerance {
		t.Errorf("size differs from du by %.2f%%, over realDirTolerance", 100*diff)
	}
}
