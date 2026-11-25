package scan

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sys/unix"
)

type flags uint8

const (
	flagDir flags = 1 << iota
	flagUnreadable
	flagLinked // regular file with more than one hard link
	flagShadow // hard link whose blocks are counted on another link
)

// Node keeps no full path: with millions of nodes the path strings would
// dominate memory, so Path rebuilds it from the parent chain.
type Node struct {
	Name     string
	Parent   *Node
	Children []*Node
	Size     int64  // disk usage of the subtree
	alloc    int64  // st_blocks * 512 of this entry alone
	Files    uint32 // files in the subtree; a file counts itself
	mtime    uint32
	newest   uint32
	flags    flags
}

func (n *Node) IsDir() bool      { return n.flags&flagDir != 0 }
func (n *Node) Unreadable() bool { return n.flags&flagUnreadable != 0 }

// Newest is the newest file mtime in the subtree, or the directory's own
// mtime when the subtree holds no files.
func (n *Node) Newest() time.Time { return time.Unix(int64(n.newest), 0) }

func (n *Node) Path() string {
	if n.Parent == nil {
		return n.Name
	}
	p := n.Parent.Path()
	if p == "/" {
		return "/" + n.Name
	}
	return p + "/" + n.Name
}

type inode struct {
	dev int32
	ino uint64
}

type Tree struct {
	Root    *Node
	dev     int32
	links   map[inode][]*Node
	linkKey map[*Node]inode
	errors  int
}

func (t *Tree) Errors() int { return t.errors }

type Options struct {
	Workers int
}

type Progress struct {
	Files   atomic.Int64
	Dirs    atomic.Int64
	Bytes   atomic.Int64
	current atomic.Pointer[string]
}

func (p *Progress) Current() string {
	if s := p.current.Load(); s != nil {
		return *s
	}
	return ""
}

func defaultWorkers() int { return 4 * runtime.NumCPU() }

func Scan(ctx context.Context, path string, opts Options, p *Progress) (*Tree, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	var st unix.Stat_t
	if err := unix.Stat(abs, &st); err != nil {
		return nil, &os.PathError{Op: "stat", Path: abs, Err: err}
	}
	if st.Mode&unix.S_IFMT != unix.S_IFDIR {
		return nil, fmt.Errorf("%s bir klasör değil", abs)
	}
	t := &Tree{
		dev:     st.Dev,
		links:   map[inode][]*Node{},
		linkKey: map[*Node]inode{},
	}
	t.Root = &Node{Name: abs}
	fill(t.Root, &st)
	w := newWalker(ctx, t.dev, opts, p)
	w.walk(t.Root, abs)
	w.wg.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	t.addLinks(w.links)
	t.resolveLinks()
	t.recompute()
	return t, nil
}

func fill(n *Node, st *unix.Stat_t) {
	n.alloc = st.Blocks * 512
	if st.Mtim.Sec > 0 {
		n.mtime = uint32(st.Mtim.Sec)
	}
	n.flags = 0
	switch {
	case st.Mode&unix.S_IFMT == unix.S_IFDIR:
		n.flags |= flagDir
	case st.Nlink > 1:
		n.flags |= flagLinked
	}
}

type walker struct {
	ctx   context.Context
	dev   int32
	sem   chan struct{}
	wg    sync.WaitGroup
	p     *Progress
	mu    sync.Mutex
	links map[*Node]inode
}

func newWalker(ctx context.Context, dev int32, opts Options, p *Progress) *walker {
	workers := opts.Workers
	if workers <= 0 {
		workers = defaultWorkers()
	}
	if p == nil {
		p = &Progress{}
	}
	return &walker{
		ctx:   ctx,
		dev:   dev,
		sem:   make(chan struct{}, workers-1),
		p:     p,
		links: map[*Node]inode{},
	}
}

func (w *walker) walk(dir *Node, path string) {
	if w.ctx.Err() != nil {
		return
	}
	w.p.Dirs.Add(1)
	w.p.current.Store(&path)

	f, err := os.Open(path)
	if err != nil {
		dir.flags |= flagUnreadable
		return
	}
	names, err := f.Readdirnames(-1)
	if err != nil {
		dir.flags |= flagUnreadable
		if len(names) == 0 {
			f.Close()
			return
		}
	}
	fd := int(f.Fd())
	nodes := make([]Node, len(names))
	dir.Children = make([]*Node, 0, len(names))
	var subdirs []*Node
	var st unix.Stat_t
	for i, name := range names {
		if err := unix.Fstatat(fd, name, &st, unix.AT_SYMLINK_NOFOLLOW); err != nil {
			if !errors.Is(err, unix.ENOENT) {
				dir.flags |= flagUnreadable
			}
			continue
		}
		n := &nodes[i]
		n.Name, n.Parent = name, dir
		fill(n, &st)
		dir.Children = append(dir.Children, n)
		w.p.Bytes.Add(n.alloc)
		switch {
		case n.IsDir():
			if st.Dev == w.dev {
				subdirs = append(subdirs, n)
			}
		default:
			w.p.Files.Add(1)
			if n.flags&flagLinked != 0 {
				w.mu.Lock()
				w.links[n] = inode{st.Dev, st.Ino}
				w.mu.Unlock()
			}
		}
	}
	// Closed before descending so open descriptors stay bounded by the
	// number of workers, not by the depth of the tree.
	f.Close()

	for _, sub := range subdirs {
		subPath := path + "/" + sub.Name
		if path == "/" {
			subPath = "/" + sub.Name
		}
		select {
		case w.sem <- struct{}{}:
			w.wg.Add(1)
			go func() {
				defer w.wg.Done()
				w.walk(sub, subPath)
				<-w.sem
			}()
		default:
			w.walk(sub, subPath)
		}
	}
}

func (t *Tree) addLinks(m map[*Node]inode) {
	for n, k := range m {
		t.links[k] = append(t.links[k], n)
		t.linkKey[n] = k
	}
}

// resolveLinks counts each hard-linked inode once, on the link with the
// smallest path, so the attribution does not depend on worker scheduling.
func (t *Tree) resolveLinks() {
	for _, nodes := range t.links {
		var owner *Node
		var ownerPath string
		for _, n := range nodes {
			n.flags |= flagShadow
			if p := n.Path(); owner == nil || p < ownerPath {
				owner, ownerPath = n, p
			}
		}
		owner.flags &^= flagShadow
	}
}

func (t *Tree) recompute() {
	t.errors = 0
	t.sum(t.Root)
}

func (t *Tree) sum(n *Node) {
	if n.Unreadable() {
		t.errors++
	}
	size := n.alloc
	if n.flags&flagShadow != 0 {
		size = 0
	}
	if !n.IsDir() {
		n.Size, n.Files, n.newest = size, 1, n.mtime
		return
	}
	var files, newest uint32
	for _, c := range n.Children {
		t.sum(c)
		size += c.Size
		files += c.Files
		if c.Files > 0 && c.newest > newest {
			newest = c.newest
		}
	}
	if files == 0 {
		newest = n.mtime
	}
	n.Size, n.Files, n.newest = size, files, newest
}

type Result struct {
	target *Node
	fresh  *Node
	links  map[*Node]inode
	gone   bool
	Err    error
}

// ScanSubtree rescans dir without touching the tree, so it can run off the UI
// goroutine; Apply then splices the result in.
func (t *Tree) ScanSubtree(ctx context.Context, dir *Node, opts Options, p *Progress) *Result {
	path := dir.Path()
	res := &Result{target: dir}
	var st unix.Stat_t
	var err error
	if dir.Parent == nil {
		err = unix.Stat(path, &st)
	} else {
		err = unix.Lstat(path, &st)
	}
	if errors.Is(err, unix.ENOENT) && dir.Parent != nil {
		res.gone = true
		return res
	}
	if err != nil {
		res.Err = &os.PathError{Op: "stat", Path: path, Err: err}
		return res
	}
	fresh := &Node{Name: dir.Name}
	fill(fresh, &st)
	w := newWalker(ctx, t.dev, opts, p)
	if fresh.IsDir() && st.Dev == t.dev {
		w.walk(fresh, path)
		w.wg.Wait()
	} else if fresh.flags&flagLinked != 0 {
		w.links[fresh] = inode{st.Dev, st.Ino}
	}
	if err := ctx.Err(); err != nil {
		res.Err = err
		return res
	}
	res.fresh, res.links = fresh, w.links
	return res
}

// Apply updates the target in place, so pointers to it and its ancestors stay
// valid; only the nodes below it are replaced. A target that is gone is
// removed from its parent instead.
func (t *Tree) Apply(res *Result) {
	if res.Err != nil {
		return
	}
	target := res.target
	t.dropLinks(target)
	if res.gone {
		parent := target.Parent
		parent.Children = slices.DeleteFunc(parent.Children, func(c *Node) bool { return c == target })
		// target shares one allocation with its siblings and lives as long as
		// they do; the subtree below it must not.
		target.Children = nil
	} else {
		fresh := res.fresh
		target.alloc, target.mtime, target.flags = fresh.alloc, fresh.mtime, fresh.flags
		target.Children = fresh.Children
		for _, c := range target.Children {
			c.Parent = target
		}
		if k, ok := res.links[fresh]; ok {
			delete(res.links, fresh)
			res.links[target] = k
		}
		t.addLinks(res.links)
	}
	t.resolveLinks()
	t.recompute()
}

func (t *Tree) dropLinks(n *Node) {
	if n.flags&flagLinked != 0 {
		k := t.linkKey[n]
		delete(t.linkKey, n)
		kept := slices.DeleteFunc(t.links[k], func(o *Node) bool { return o == n })
		if len(kept) == 0 {
			delete(t.links, k)
		} else {
			t.links[k] = kept
		}
	}
	for _, c := range n.Children {
		t.dropLinks(c)
	}
}
