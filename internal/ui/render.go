package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/lemiorhan/dirstato/internal/scan"
	"github.com/lemiorhan/dirstato/internal/units"
)

// Below these terminal widths the date column, then the count column, is
// dropped so the name keeps room.
const (
	dropDateBelow  = 90
	dropCountBelow = 70
	// The full key legend from the approved mockup needs this many rows;
	// shorter terminals get a one-line legend.
	fullLegendMinHeight = 30
	barWidth            = 10
)

var (
	selectedStyle = lipgloss.NewStyle().Reverse(true)
	headerStyle   = lipgloss.NewStyle().Bold(true)
	dimStyle      = lipgloss.NewStyle().Faint(true)
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
)

var fullLegend = []string{
	"↑↓ j k   hareket        Tab  tree ↔ drill-down",
	"→ l ⏎   aç / içeri gir  s    size desc→asc→isim",
	"← h     kapat / çık     m    minimum boyut gir",
	"c       path kopyala     r    seçili klasörü tara",
	"?       yardım           q    çık",
}

const shortLegend = "↑↓ hareket · →← aç/kapat · Tab görünüm · s sırala · m min · c kopyala · r tara · ? yardım · q çık"

var helpText = []string{
	"Tuşlar",
	"  ↑ ↓ j k          bir satır yukarı / aşağı",
	"  pgup pgdown      bir sayfa yukarı / aşağı",
	"  home g / end G   listenin başı / sonu",
	"  → l ⏎            tree'de klasörü açar; açıksa ilk çocuğa iner. drill-down'da klasöre girer",
	"  ← h              tree'de klasörü kapatır ya da parent'a çıkar. drill-down'da bir üst klasöre çıkar",
	"  Tab              tree ↔ drill-down; seçili öğe korunur",
	"  s                sıralama: size desc → size asc → isim",
	"  m                minimum boyut gir (500M, 1.5G, 0 = kapalı)",
	"  c                seçili öğenin absolute path'ini kopyalar (gerekirse tek tırnakla)",
	"  r                seçili klasörü yeniden tarar; dosyada parent klasörü tarar",
	"  ?                bu ekranı açar / kapatır",
	"  q ctrl+c         çıkış",
	"",
	"Kolonlar",
	"  boyut            disk kullanımı, du -skx ile aynı ölçü; 1024 tabanı",
	"  %, bar           parent klasörün içindeki payı",
	"  sayı             alt tree'deki dosya sayısı",
	"  tarih            alt tree'deki en yeni dosyanın mtime'ı",
	"  … küçük öğe      minimum boyutun altında kalıp gizlenen öğelerin toplamı",
	"  !                okunamayan klasör; boyutu eksik olabilir",
	"",
	helpClose,
}

const helpClose = "Kapatmak için ? ya da esc."

func (m Model) render() string {
	switch m.state {
	case scanning:
		return m.renderProgress()
	}
	if m.help {
		lines := append([]string{headerStyle.Render("dirstato · yardım")}, helpText...)
		if len(lines) > m.height {
			lines = append(lines[:max(0, m.height-1)], helpClose)
		}
		return m.fit(lines)
	}
	lines := append(m.header(), strings.Repeat("─", m.width))
	h := m.listHeight()
	end := min(m.offset+h, len(m.rows))
	for i := m.offset; i < end; i++ {
		if i == m.cursor {
			// Unstyled: an inner style's reset would end the reverse video mid-line.
			line := m.renderRow(m.rows[i], false)
			lines = append(lines, selectedStyle.Render(line+strings.Repeat(" ", max(0, m.width-ansi.StringWidth(line)))))
			continue
		}
		lines = append(lines, m.renderRow(m.rows[i], true))
	}
	if len(m.rows) == 0 {
		lines = append(lines, dimStyle.Render("  (boş)"))
		end++
	}
	for i := end - m.offset; i < h; i++ {
		lines = append(lines, "")
	}
	lines = append(lines, strings.Repeat("─", m.width))
	lines = append(lines, m.legend()...)
	lines = append(lines, m.statusLine())
	return strings.Join(lines, "\n")
}

func (m Model) fit(lines []string) string {
	for i, l := range lines {
		lines[i] = ansi.Truncate(l, m.width, "…")
	}
	if len(lines) > m.height {
		lines = lines[:m.height]
	}
	return strings.Join(lines, "\n")
}

func (m Model) legend() []string {
	if m.height >= fullLegendMinHeight {
		return fullLegend
	}
	return []string{ansi.Truncate(shortLegend, m.width, "…")}
}

func (m Model) listHeight() int {
	chrome := 4 // header, two separators, status line
	if m.height >= fullLegendMinHeight {
		chrome += len(fullLegend)
	} else {
		chrome++
	}
	if m.view == drillView {
		chrome++ // current folder line under the header
	}
	return max(1, m.height-chrome)
}

func (m Model) header() []string {
	view := "tree"
	if m.view == drillView {
		view = "drill-down"
	}
	minLabel := "min yok"
	if m.min > 0 {
		minLabel = "min " + m.minText
	}
	labels := " · " + sortLabels[m.sort] + " · " + minLabel + " · " + view
	if n := m.tree.Errors(); n > 0 {
		labels += " · " + errorStyle.Render(fmt.Sprintf("%d okunamayan klasör", n))
	}
	title := headerStyle.Render("dirstato") + "  "
	room := m.width - ansi.StringWidth(title) - ansi.StringWidth(labels)
	lines := []string{ansi.Truncate(title+fitLeft(m.display(m.tree.Root), room)+labels, m.width, "…")}
	if m.view == drillView {
		lines = append(lines, dimStyle.Render("▸ "+fitLeft(m.display(m.cwd), m.width-2)))
	}
	return lines
}

func (m Model) display(n *scan.Node) string {
	p := n.Path()
	if m.home != "" && (p == m.home || strings.HasPrefix(p, m.home+"/")) {
		return "~" + p[len(m.home):]
	}
	return p
}

// fitLeft keeps the end of a path, which is the part that tells folders apart.
func fitLeft(s string, w int) string {
	if over := ansi.StringWidth(s) - w; over > 0 && w > 1 {
		return ansi.TruncateLeft(s, over+1, "…")
	}
	return s
}

func (m Model) statusLine() string {
	switch {
	case m.editing:
		return m.input.View() + dimStyle.Render("   ⏎ uygula · esc vazgeç")
	case m.state == rescanning:
		return fmt.Sprintf("yeniden taranıyor… %s dosya · %s · %s",
			formatCount(m.progress.Files.Load()), units.FormatSize(m.progress.Bytes.Load()),
			ansi.Truncate(m.progress.Current(), max(10, m.width-50), "…"))
	}
	return ansi.Truncate(m.status, m.width, "…")
}

func (m Model) renderProgress() string {
	p := m.progress
	elapsed := time.Since(m.started).Round(time.Second)
	lines := []string{
		headerStyle.Render("dirstato") + "  " + m.cfg.Root,
		"",
		fmt.Sprintf("taranıyor… %s dosya · %s klasör · %s · %s",
			formatCount(p.Files.Load()), formatCount(p.Dirs.Load()), units.FormatSize(p.Bytes.Load()), elapsed),
		dimStyle.Render(p.Current()),
		"",
		dimStyle.Render("q / ctrl+c: iptal"),
	}
	return m.fit(lines)
}

func (m Model) renderRow(r row, styled bool) string {
	dim, bad := dimStyle.Render, errorStyle.Render
	if !styled {
		dim = func(s ...string) string { return strings.Join(s, "") }
		bad = dim
	}
	showDate := m.width >= dropDateBelow
	showCount := m.width >= dropCountBelow

	var size, files int64
	var newest time.Time
	parent := r.parent
	if r.node != nil {
		size, files, newest = r.node.Size, int64(r.node.Files), r.node.Newest()
	} else {
		size, files = r.hiddenSize, r.hiddenFiles
	}
	pct := 100.0
	if parent != nil {
		pct = 0
		if parent.Size > 0 {
			pct = 100 * float64(size) / float64(parent.Size)
		}
	}
	cols := fmt.Sprintf("%7s %3.0f%% %-*s", units.FormatSize(size), pct, barWidth, bar(pct))
	if showCount {
		cols += fmt.Sprintf(" %6s", formatCount(files))
	}
	if showDate {
		date := ""
		if r.node != nil {
			date = newest.Format("2006-01-02")
		}
		cols += fmt.Sprintf(" %10s", date)
	}
	indent := ""
	if m.view == treeView {
		indent = strings.Repeat("  ", r.depth)
	}
	var name string
	switch {
	case r.node == nil:
		name = dim(fmt.Sprintf("  … %s küçük öğe", formatCount(int64(r.hiddenCount))))
	case r.node.IsDir():
		marker := "▸ "
		if m.view == treeView && m.expanded[r.node] {
			marker = "▾ "
		}
		label := r.node.Name + "/"
		if r.node.Parent == nil {
			label = m.display(r.node)
		}
		name = marker + label
		if r.node.Unreadable() {
			name += " " + bad("!")
		}
	default:
		name = "  " + r.node.Name
	}
	line := cols + " " + indent + name
	return ansi.Truncate(line, m.width, "…")
}

func bar(pct float64) string {
	n := int(math.Round(pct / 100 * barWidth))
	if n == 0 && pct > 0 {
		n = 1
	}
	return strings.Repeat("█", min(n, barWidth))
}

func formatCount(n int64) string {
	switch {
	case n < 1000:
		return fmt.Sprintf("%d", n)
	case n < 10_000:
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	case n < 1_000_000:
		return fmt.Sprintf("%.0fK", float64(n)/1000)
	case n < 10_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1e6)
	default:
		return fmt.Sprintf("%.0fM", float64(n)/1e6)
	}
}
