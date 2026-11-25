package ui

import (
	"context"
	"errors"
	"os"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/lemiorhan/dirstato/internal/clip"
	"github.com/lemiorhan/dirstato/internal/scan"
	"github.com/lemiorhan/dirstato/internal/units"
)

type Config struct {
	Root    string
	Min     int64
	MinText string // Min as typed after --min; the header shows this text
	Copy    func(string) error
}

type state int

const (
	scanning state = iota
	browsing
	rescanning
)

type viewMode int

const (
	treeView viewMode = iota
	drillView
)

type sortMode int

const (
	sizeDesc sortMode = iota
	sizeAsc
	byName
)

var sortLabels = []string{"size desc", "size asc", "name"}

type row struct {
	node        *scan.Node // nil on a summary row
	parent      *scan.Node
	depth       int
	hiddenCount int
	hiddenSize  int64
	hiddenFiles int64
}

func (r row) target() *scan.Node {
	if r.node != nil {
		return r.node
	}
	return r.parent
}

type Model struct {
	cfg      Config
	state    state
	tree     *scan.Tree
	progress *scan.Progress
	cancel   context.CancelFunc
	startCmd tea.Cmd
	err      error
	started  time.Time

	view     viewMode
	sort     sortMode
	min      int64
	minText  string // a formatted min could round away from what was typed
	expanded map[*scan.Node]bool
	cwd      *scan.Node
	rows     []row
	cursor   int
	offset   int

	width, height int
	home          string
	status        string
	statusID      int
	help          bool
	editing       bool
	input         textinput.Model
}

type (
	tickMsg     struct{}
	scanDoneMsg struct {
		tree *scan.Tree
		err  error
	}
	rescanDoneMsg struct{ res *scan.Result }
	clearStatus   struct{ id int }
)

const (
	tickEvery     = 100 * time.Millisecond
	statusTimeout = 3 * time.Second
)

func New(cfg Config) Model {
	in := textinput.New()
	in.Prompt = "Minimum size: "
	in.Placeholder = "500M, 1.5G, 0"
	ctx, cancel := context.WithCancel(context.Background())
	p := &scan.Progress{}
	home, _ := os.UserHomeDir()
	return Model{
		home:     home,
		cfg:      cfg,
		cancel:   cancel,
		progress: p,
		started:  time.Now(),
		startCmd: func() tea.Msg {
			t, err := scan.Scan(ctx, cfg.Root, scan.Options{}, p)
			return scanDoneMsg{t, err}
		},
		min:      cfg.Min,
		minText:  cfg.MinText,
		expanded: map[*scan.Node]bool{},
		input:    in,
		width:    80,
		height:   24,
	}
}

func (m Model) Err() error { return m.err }

func tick() tea.Cmd {
	return tea.Tick(tickEvery, func(time.Time) tea.Msg { return tickMsg{} })
}

func (m Model) Init() tea.Cmd { return tea.Batch(tick(), m.startCmd) }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m.update(msg) }

func (m Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	return v
}

func (m Model) update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.scrollToCursor()
		return m, nil
	case tickMsg:
		if m.state == browsing {
			return m, nil
		}
		return m, tick()
	case scanDoneMsg:
		if msg.err != nil {
			if !errors.Is(msg.err, context.Canceled) {
				m.err = msg.err
			}
			return m, tea.Quit
		}
		m.tree, m.state = msg.tree, browsing
		m.expanded[m.tree.Root] = true
		m.cwd = m.tree.Root
		m.rebuild()
		return m, nil
	case rescanDoneMsg:
		m.state = browsing
		if msg.res.Err != nil {
			if errors.Is(msg.res.Err, context.Canceled) {
				return m, nil
			}
			return m.flash("Rescan failed: " + msg.res.Err.Error())
		}
		sel, hadSel := m.selected()
		m.tree.Apply(msg.res)
		m.cwd, _ = relocate(m.cwd)
		if p := m.cwd.Parent; !m.cwd.IsDir() && p != nil {
			m.cwd = p
		}
		expanded := make(map[*scan.Node]bool, len(m.expanded))
		for n := range m.expanded {
			if r, exact := relocate(n); exact && r.IsDir() {
				expanded[r] = true
			}
		}
		m.expanded = expanded
		m.rebuild()
		if hadSel {
			n, exact := relocate(sel.target())
			if sel.node != nil || !exact || !m.selectSummary(n) {
				m.selectNearest(n)
			}
		}
		return m, nil
	case clearStatus:
		if msg.id == m.statusID {
			m.status = ""
		}
		return m, nil
	case tea.KeyPressMsg:
		return m.key(msg)
	}
	if m.editing {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) key(k tea.KeyPressMsg) (Model, tea.Cmd) {
	key := k.String()
	if key == "ctrl+c" {
		if m.cancel != nil {
			m.cancel()
		}
		return m, tea.Quit
	}
	if m.state != browsing {
		if key == "q" {
			if m.cancel != nil {
				m.cancel()
			}
			return m, tea.Quit
		}
		return m, nil
	}
	if m.editing {
		switch key {
		case "enter":
			m.editing = false
			m.input.Blur()
			v, err := units.ParseSize(m.input.Value())
			if err != nil {
				return m.flash(err.Error())
			}
			m.min, m.minText = v, strings.TrimSpace(m.input.Value())
			m.rebuild()
			return m, nil
		case "esc":
			m.editing = false
			m.input.Blur()
			return m, nil
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(k)
		return m, cmd
	}
	if m.help {
		switch key {
		case "?", "esc", "q":
			m.help = false
		}
		return m, nil
	}

	switch key {
	case "q":
		return m, tea.Quit
	case "up", "k":
		m.move(-1)
	case "down", "j":
		m.move(1)
	case "pgup":
		m.move(-m.listHeight())
	case "pgdown":
		m.move(m.listHeight())
	case "home", "g":
		m.move(-len(m.rows))
	case "end", "G":
		m.move(len(m.rows))
	case "right", "l", "enter":
		m.open()
	case "left", "h":
		m.close()
	case "tab":
		m.toggleView()
	case "s":
		m.sort = (m.sort + 1) % sortMode(len(sortLabels))
		m.rebuild()
	case "m":
		m.editing = true
		if m.min > 0 {
			m.input.SetValue(m.minText)
		} else {
			m.input.SetValue("")
		}
		m.input.CursorEnd()
		return m, m.input.Focus()
	case "c":
		return m.copySelected()
	case "r":
		return m.rescanSelected()
	case "?":
		m.help = true
	}
	return m, nil
}

func (m Model) flash(s string) (Model, tea.Cmd) {
	m.statusID++
	m.status = s
	id := m.statusID
	return m, tea.Tick(statusTimeout, func(time.Time) tea.Msg { return clearStatus{id} })
}

func (m *Model) selected() (row, bool) {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return row{}, false
	}
	return m.rows[m.cursor], true
}

func (m Model) copySelected() (Model, tea.Cmd) {
	r, ok := m.selected()
	if !ok {
		return m, nil
	}
	text := clip.Quote(r.target().Path())
	if err := m.cfg.Copy(text); err != nil {
		return m.flash("Copy failed: " + err.Error())
	}
	return m.flash("Copied: " + text)
}

func (m Model) rescanSelected() (Model, tea.Cmd) {
	dir := m.cwd
	if r, ok := m.selected(); ok {
		dir = r.target()
		if !dir.IsDir() {
			dir = dir.Parent
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.progress = &scan.Progress{}
	m.started = time.Now()
	m.state = rescanning
	tree, p := m.tree, m.progress
	return m, tea.Batch(tick(), func() tea.Msg {
		return rescanDoneMsg{tree.ScanSubtree(ctx, dir, scan.Options{}, p)}
	})
}

// relocate maps a node from before a rescan to the node now at its path. Apply
// replaces everything under the rescanned folder, and holding on to an old node
// would also keep the whole replaced subtree in memory. When the path is gone,
// relocate returns its deepest surviving ancestor and exact is false.
func relocate(n *scan.Node) (found *scan.Node, exact bool) {
	var names []string
	for ; !attached(n); n = n.Parent {
		names = append(names, n.Name)
	}
	for i := len(names) - 1; i >= 0; i-- {
		next := childNamed(n, names[i])
		if next == nil {
			return n, false
		}
		n = next
	}
	return n, true
}

func childNamed(n *scan.Node, name string) *scan.Node {
	for _, c := range n.Children {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func attached(n *scan.Node) bool {
	for ; n.Parent != nil; n = n.Parent {
		found := false
		for _, c := range n.Parent.Children {
			if c == n {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (m *Model) move(d int) {
	m.cursor += d
	m.clampCursor()
}

func (m *Model) clampCursor() {
	if m.cursor >= len(m.rows) {
		m.cursor = len(m.rows) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.scrollToCursor()
}

func (m *Model) scrollToCursor() {
	h := m.listHeight()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+h {
		m.offset = m.cursor - h + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

func (m *Model) open() {
	r, ok := m.selected()
	if !ok {
		return
	}
	if r.node == nil {
		// → on a summary row acts on its folder, which is already open (tree) or
		// entered (drill-down), so the cursor goes to that folder's first row.
		if m.view == treeView {
			m.selectNode(r.parent)
			m.move(1)
		} else {
			m.move(-len(m.rows))
		}
		return
	}
	if !r.node.IsDir() {
		return
	}
	if m.view == drillView {
		m.cwd = r.node
		m.cursor, m.offset = 0, 0
		m.rebuild()
		return
	}
	if !m.expanded[r.node] {
		m.expanded[r.node] = true
		m.rebuild()
		return
	}
	if m.cursor+1 < len(m.rows) && m.rows[m.cursor+1].depth > r.depth {
		m.move(1)
	}
}

func (m *Model) close() {
	if m.view == drillView {
		if m.cwd.Parent == nil {
			return
		}
		from := m.cwd
		m.cwd = m.cwd.Parent
		m.rebuild()
		m.selectNearest(from)
		return
	}
	r, ok := m.selected()
	if !ok {
		return
	}
	if r.node != nil && r.node.IsDir() && m.expanded[r.node] {
		delete(m.expanded, r.node)
		m.rebuild()
		return
	}
	parent := r.parent
	if r.node != nil {
		parent = r.node.Parent
	}
	if parent != nil {
		m.selectNode(parent)
	}
}

func (m *Model) toggleView() {
	r, ok := m.selected()
	if m.view == treeView {
		m.view = drillView
		switch {
		case !ok:
			m.cwd = m.tree.Root
		case r.node == nil:
			m.cwd = r.parent
		case r.node.Parent == nil:
			m.cwd = r.node
		default:
			m.cwd = r.node.Parent
		}
		m.rebuild()
		if ok && r.node == nil {
			m.selectSummary(r.parent)
		} else if ok && r.node.Parent != nil {
			m.selectNode(r.node)
		} else {
			m.cursor, m.offset = 0, 0
			m.clampCursor()
		}
		return
	}
	m.view = treeView
	target := m.cwd
	if ok {
		target = r.target()
	}
	for a := target.Parent; a != nil; a = a.Parent {
		m.expanded[a] = true
	}
	if ok && r.node == nil {
		m.expanded[r.parent] = true
		m.rebuild()
		m.selectSummary(r.parent)
		return
	}
	m.rebuild()
	m.selectNearest(target)
}

func (m *Model) selectNode(n *scan.Node) bool {
	for i, r := range m.rows {
		if r.node == n {
			m.cursor = i
			m.scrollToCursor()
			return true
		}
	}
	return false
}

func (m *Model) selectSummary(parent *scan.Node) bool {
	for i, r := range m.rows {
		if r.node == nil && r.parent == parent {
			m.cursor = i
			m.scrollToCursor()
			return true
		}
	}
	return false
}

// selectNearest selects the row standing for n: its own row, else the summary
// row hiding it, else the same for its nearest ancestor. In drill-down that
// can be the current folder, which has no row of its own; the cursor then goes
// to the first row, as when entering the folder.
func (m *Model) selectNearest(n *scan.Node) {
	for ; n != nil; n = n.Parent {
		if m.selectNode(n) || n.Parent != nil && m.selectSummary(n.Parent) {
			return
		}
	}
	m.move(-len(m.rows))
}

func (m *Model) rebuild() {
	prev, hadPrev := m.selected()
	// A fresh slice: reusing the old one leaves rows past the new length in its
	// spare capacity, and those keep nodes a rescan replaced alive.
	m.rows = make([]row, 0, len(m.rows))
	if m.view == treeView {
		root := m.tree.Root
		m.rows = append(m.rows, row{node: root, depth: 0})
		if m.expanded[root] {
			m.addChildren(root, 1, true)
		}
	} else {
		m.addChildren(m.cwd, 0, false)
	}
	if hadPrev {
		for i, r := range m.rows {
			if r.node == prev.node && r.parent == prev.parent {
				m.cursor = i
				m.clampCursor()
				return
			}
		}
		m.selectNearest(prev.target())
		return
	}
	m.clampCursor()
}

func (m *Model) addChildren(n *scan.Node, depth int, recurse bool) {
	m.sortChildren(n)
	hidden := row{parent: n, depth: depth}
	for _, c := range n.Children {
		if c.Size < m.min {
			hidden.hiddenCount++
			hidden.hiddenSize += c.Size
			hidden.hiddenFiles += int64(c.Files)
			continue
		}
		m.rows = append(m.rows, row{node: c, parent: n, depth: depth})
		if recurse && c.IsDir() && m.expanded[c] {
			m.addChildren(c, depth+1, true)
		}
	}
	if hidden.hiddenCount > 0 {
		m.rows = append(m.rows, hidden)
	}
}

func (m *Model) sortChildren(n *scan.Node) {
	byNameLess := func(a, b *scan.Node) bool {
		la, lb := strings.ToLower(a.Name), strings.ToLower(b.Name)
		if la != lb {
			return la < lb
		}
		return a.Name < b.Name
	}
	kids := n.Children
	switch m.sort {
	case sizeDesc:
		sort.SliceStable(kids, func(i, j int) bool {
			if kids[i].Size != kids[j].Size {
				return kids[i].Size > kids[j].Size
			}
			return byNameLess(kids[i], kids[j])
		})
	case sizeAsc:
		sort.SliceStable(kids, func(i, j int) bool {
			if kids[i].Size != kids[j].Size {
				return kids[i].Size < kids[j].Size
			}
			return byNameLess(kids[i], kids[j])
		})
	case byName:
		sort.SliceStable(kids, func(i, j int) bool { return byNameLess(kids[i], kids[j]) })
	}
}
