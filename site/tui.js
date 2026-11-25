(function (g) {
  "use strict";

  const DROP_DATE_BELOW = 90;
  const DROP_COUNT_BELOW = 70;
  const FULL_LEGEND_MIN_HEIGHT = 30;
  const BAR_WIDTH = 10;
  const STATUS_TIMEOUT = 3000;

  const FULL_LEGEND = [
    "↑↓ j k   move           Tab  tree ↔ drill-down",
    "→ l ⏎    open / enter   s    size desc→asc→name",
    "← h      close / back   m    set minimum size",
    "c        copy path      r    rescan selection",
    "?        help           q    quit",
  ];
  const SHORT_LEGEND = "→← open/close · Tab view · s sort · m min · c copy · r rescan · ? help · q quit";
  const HELP_CLOSE = "Press ? or esc to close.";
  const HELP_TEXT = [
    "Keys",
    "  ↑ ↓ j k          move up / down one row",
    "  pgup pgdown      move up / down one page",
    "  home g / end G   jump to the first / last row",
    "  → l ⏎            tree: expand or go to first child · drill-down: enter",
    "  ← h              tree: collapse or go to parent · drill-down: go up",
    "  Tab              switch tree ↔ drill-down, keeping the selection",
    "  s                sort: size desc → size asc → name",
    "  m                set the minimum size (500M, 1.5G, 0 = off)",
    "  c                copy the absolute path, shell-quoted when needed",
    "  r                rescan the selected folder (on a file: its parent)",
    "  ?                open / close this screen",
    "  q ctrl+c         quit",
    "",
    "Columns",
    "  size             disk usage, the same measure as du -skx; base 1024",
    "  %, bar           share of the parent folder",
    "  count            number of files in the subtree",
    "  date             newest file modification time in the subtree",
    "  … small items    total of the items hidden below the minimum size",
    "  !                unreadable folder; its size may be incomplete",
    "",
    HELP_CLOSE,
  ];
  const SORT_LABELS = ["size desc", "size asc", "name"];
  const SUFFIXES = ["B", "K", "M", "G", "T", "P"];

  function roundHalfEven(x) {
    const f = Math.floor(x);
    const d = x - f;
    if (d > 0.5) return f + 1;
    if (d < 0.5) return f;
    return f % 2 === 0 ? f : f + 1;
  }

  // A double sits exactly halfway between two tenths only when 4v is an odd
  // integer; v * 10 is itself rounded, so testing it finds false ties.
  function fixed1(v) {
    const quarters = v * 4;
    if (Number.isInteger(quarters) && quarters % 2 !== 0) {
      return (roundHalfEven(v * 10) / 10).toFixed(1);
    }
    return v.toFixed(1);
  }

  function formatSize(n) {
    if (n < 1024) return n + "B";
    let v = n;
    let i = 0;
    while (v >= 1024 && i < SUFFIXES.length - 1) {
      v /= 1024;
      i++;
    }
    if (fixed1(v) === "1024.0" && i < SUFFIXES.length - 1) {
      v /= 1024;
      i++;
    }
    return fixed1(v) + SUFFIXES[i];
  }

  function formatCount(n) {
    if (n < 1000) return String(n);
    if (n < 10000) return fixed1(n / 1000) + "K";
    if (n < 1e6) return roundHalfEven(n / 1000) + "K";
    if (n < 1e7) return fixed1(n / 1e6) + "M";
    return roundHalfEven(n / 1e6) + "M";
  }

  const SIZE_PATTERN = /^(\d+(?:\.\d+)?)[\t\n\f\r ]*([kmgtp]?)(?:i?b)?$/;

  function parseSize(s) {
    const m = SIZE_PATTERN.exec(s.trim().toLowerCase());
    if (!m) throw new Error(`invalid size: ${JSON.stringify(s)} (examples: 500M, 1.5G, 0)`);
    const exp = m[2] === "" ? 0 : "kmgtp".indexOf(m[2]) + 1;
    const bytes = parseFloat(m[1]) * Math.pow(1024, exp);
    if (bytes >= 2 ** 63) throw new Error(`size too large: ${JSON.stringify(s)}`);
    return Math.trunc(bytes);
  }

  function plural(n, word) {
    return n === 1 ? word : word + "s";
  }

  function bar(pct) {
    let n = Math.round((pct / 100) * BAR_WIDTH);
    if (n === 0 && pct > 0) n = 1;
    return "█".repeat(Math.min(n, BAR_WIDTH));
  }

  function quote(s) {
    if (s === "") return "''";
    if (!/[^A-Za-z0-9_@%+=:,./-]/.test(s)) return s;
    return "'" + s.replace(/'/g, "'\\''") + "'";
  }

  function padStart(s, w) {
    return s.length >= w ? s : " ".repeat(w - s.length) + s;
  }

  function seg(t, s) {
    return { t: t, s: s || "" };
  }

  function segWidth(segs) {
    return segs.reduce((w, x) => w + x.t.length, 0);
  }

  function truncate(segs, w) {
    if (segWidth(segs) <= w) return segs;
    const out = [];
    let room = Math.max(0, w - 1);
    for (const x of segs) {
      if (room <= 0) break;
      out.push(seg(x.t.slice(0, room), x.s));
      room -= Math.min(room, x.t.length);
    }
    out.push(seg("…"));
    return out;
  }

  function fitLeft(s, w) {
    const over = s.length - w;
    if (over > 0 && w > 1) return "…" + s.slice(over + 1);
    return s;
  }

  function prepare(node, parent) {
    node.p = parent;
    if (node.c) for (const c of node.c) prepare(c, node);
    return node;
  }

  function pathOf(n, rootPath) {
    const names = [];
    for (let x = n; x.p; x = x.p) names.unshift(x.n);
    return names.length ? rootPath + "/" + names.join("/") : rootPath;
  }

  function byNameLess(a, b) {
    const la = a.n.toLowerCase();
    const lb = b.n.toLowerCase();
    if (la !== lb) return la < lb ? -1 : 1;
    return a.n < b.n ? -1 : a.n > b.n ? 1 : 0;
  }

  class Model {
    constructor(scan, opts) {
      this.meta = scan.meta;
      this.floorText = formatSize(scan.meta.floor).replace(".0", "");
      this.root = prepare(scan.root, null);
      this.root.n = scan.meta.path;
      this.command = opts.command || "dirstato";
      this.width = opts.width || 80;
      this.height = opts.height || 24;
      this.view = "tree";
      this.sort = 0;
      this.min = opts.min || 0;
      this.minText = opts.minText || "";
      this.expanded = new Set([this.root]);
      this.cwd = this.root;
      this.rows = [];
      this.cursor = 0;
      this.offset = 0;
      this.status = "";
      this.statusID = 0;
      this.help = false;
      this.editing = false;
      this.input = "";
      this.state = "browsing";
      this.progress = { files: 0, dirs: 0, bytes: 0, current: "", elapsed: "0s" };
      this.rebuild();
    }

    path(n) {
      return pathOf(n, this.meta.path);
    }

    isDir(n) {
      return n.x === true;
    }

    selected() {
      return this.rows[this.cursor] || null;
    }

    target(r) {
      return r.node || r.parent;
    }

    flash(s) {
      this.statusID++;
      this.status = s;
      return { clearStatus: this.statusID, after: STATUS_TIMEOUT };
    }

    clearStatus(id) {
      if (id === this.statusID) this.status = "";
    }

    listHeight() {
      let chrome = 4;
      chrome += this.height >= FULL_LEGEND_MIN_HEIGHT ? FULL_LEGEND.length : 1;
      if (this.view === "drill") chrome++;
      return Math.max(1, this.height - chrome);
    }

    move(d) {
      this.cursor += d;
      this.clampCursor();
    }

    clampCursor() {
      if (this.cursor >= this.rows.length) this.cursor = this.rows.length - 1;
      if (this.cursor < 0) this.cursor = 0;
      this.scrollToCursor();
    }

    scrollToCursor() {
      const h = this.listHeight();
      if (this.cursor < this.offset) this.offset = this.cursor;
      if (this.cursor >= this.offset + h) this.offset = this.cursor - h + 1;
      if (this.offset < 0) this.offset = 0;
    }

    sortChildren(n) {
      if (!n.c) return;
      if (this.sort === 0) n.c.sort((a, b) => (a.s !== b.s ? b.s - a.s : byNameLess(a, b)));
      else if (this.sort === 1) n.c.sort((a, b) => (a.s !== b.s ? a.s - b.s : byNameLess(a, b)));
      else n.c.sort(byNameLess);
    }

    addChildren(n, depth, recurse) {
      this.sortChildren(n);
      const hidden = { node: null, parent: n, depth: depth, hiddenCount: 0, hiddenSize: 0, hiddenFiles: 0 };
      for (const c of n.c || []) {
        if (c.s < this.min) {
          hidden.hiddenCount++;
          hidden.hiddenSize += c.s;
          hidden.hiddenFiles += c.f;
          continue;
        }
        this.rows.push({ node: c, parent: n, depth: depth });
        if (recurse && this.isDir(c) && this.expanded.has(c)) this.addChildren(c, depth + 1, true);
      }
      if (n.r) {
        hidden.hiddenCount += n.r[0];
        hidden.hiddenSize += n.r[1];
        hidden.hiddenFiles += n.r[2];
      }
      if (hidden.hiddenCount > 0) this.rows.push(hidden);
    }

    rebuild() {
      const prev = this.selected();
      this.rows = [];
      if (this.view === "tree") {
        this.rows.push({ node: this.root, parent: null, depth: 0 });
        if (this.expanded.has(this.root)) this.addChildren(this.root, 1, true);
      } else {
        this.addChildren(this.cwd, 0, false);
      }
      if (prev) {
        for (let i = 0; i < this.rows.length; i++) {
          const r = this.rows[i];
          if (r.node === prev.node && r.parent === prev.parent) {
            this.cursor = i;
            this.clampCursor();
            return;
          }
        }
        this.selectNearest(this.target(prev));
        return;
      }
      this.clampCursor();
    }

    selectNode(n) {
      for (let i = 0; i < this.rows.length; i++) {
        if (this.rows[i].node === n) {
          this.cursor = i;
          this.scrollToCursor();
          return true;
        }
      }
      return false;
    }

    selectSummary(parent) {
      for (let i = 0; i < this.rows.length; i++) {
        const r = this.rows[i];
        if (r.node === null && r.parent === parent) {
          this.cursor = i;
          this.scrollToCursor();
          return true;
        }
      }
      return false;
    }

    selectNearest(n) {
      for (; n; n = n.p) {
        if (this.selectNode(n) || (n.p && this.selectSummary(n.p))) return;
      }
      this.move(-this.rows.length);
    }

    open() {
      const r = this.selected();
      if (!r) return null;
      if (r.node === null) {
        if (this.view === "tree") {
          this.selectNode(r.parent);
          this.move(1);
        } else {
          this.move(-this.rows.length);
        }
        return null;
      }
      if (!this.isDir(r.node)) return null;
      if (this.view === "drill") {
        this.cwd = r.node;
        this.cursor = 0;
        this.offset = 0;
        this.rebuild();
        return "entered";
      }
      if (!this.expanded.has(r.node)) {
        this.expanded.add(r.node);
        this.rebuild();
        return "expanded";
      }
      if (this.cursor + 1 < this.rows.length && this.rows[this.cursor + 1].depth > r.depth) this.move(1);
      return null;
    }

    close() {
      if (this.view === "drill") {
        if (!this.cwd.p) return null;
        const from = this.cwd;
        this.cwd = this.cwd.p;
        this.rebuild();
        this.selectNearest(from);
        return "up";
      }
      const r = this.selected();
      if (!r) return null;
      if (r.node && this.isDir(r.node) && this.expanded.has(r.node)) {
        this.expanded.delete(r.node);
        this.rebuild();
        return "collapsed";
      }
      const parent = r.node ? r.node.p : r.parent;
      if (parent) this.selectNode(parent);
      return null;
    }

    toggleView() {
      const r = this.selected();
      if (this.view === "tree") {
        this.view = "drill";
        if (!r) this.cwd = this.root;
        else if (r.node === null) this.cwd = r.parent;
        else if (!r.node.p) this.cwd = r.node;
        else this.cwd = r.node.p;
        this.rebuild();
        if (r && r.node === null) this.selectSummary(r.parent);
        else if (r && r.node.p) this.selectNode(r.node);
        else {
          this.cursor = 0;
          this.offset = 0;
          this.clampCursor();
        }
        return;
      }
      this.view = "tree";
      const target = r ? this.target(r) : this.cwd;
      for (let a = target.p; a; a = a.p) this.expanded.add(a);
      if (r && r.node === null) {
        this.expanded.add(r.parent);
        this.rebuild();
        this.selectSummary(r.parent);
        return;
      }
      this.rebuild();
      this.selectNearest(target);
    }

    rescanTarget() {
      let dir = this.cwd;
      const r = this.selected();
      if (r) {
        dir = this.target(r);
        if (!this.isDir(dir)) dir = dir.p;
      }
      return dir;
    }

    applyMin(text) {
      let v;
      try {
        v = parseSize(text);
      } catch (e) {
        return { effect: this.flash(e.message) };
      }
      let label = text.trim();
      let effect = null;
      if (v < this.meta.floor) {
        v = this.meta.floor;
        label = this.floorText;
        effect = this.flash("This demo scan keeps items of " + label + " or more, so the minimum is " + label + ".");
      }
      this.min = v;
      this.minText = label;
      this.rebuild();
      return { effect: effect, applied: true };
    }

    key(key) {
      if (this.state === "launch") return key === "enter" ? { action: "run" } : {};
      if (this.state === "quit") {
        if (key === "enter" || key === "r") return { action: "restart" };
        return {};
      }
      if (key === "ctrl+c") return { action: "quit" };
      if (this.state !== "browsing") {
        if (key === "q") return { action: "quit" };
        return {};
      }
      if (this.editing) {
        if (key === "enter") {
          this.editing = false;
          const res = this.applyMin(this.input);
          return { effect: res.effect, did: res.applied ? "min" : null };
        }
        if (key === "esc") {
          this.editing = false;
          return {};
        }
        if (key === "backspace") {
          this.input = this.input.slice(0, -1);
          return {};
        }
        if (key.length === 1) this.input += key;
        return {};
      }
      if (this.help) {
        if (key === "?" || key === "esc" || key === "q") this.help = false;
        return { did: this.help ? null : "help-close" };
      }
      const before = this.cursor;
      const moved = (did) => ({ did: did || (this.cursor !== before ? "move" : "nothing") });
      switch (key) {
        case "q":
          return { action: "quit" };
        case "up":
        case "k":
          this.move(-1);
          return moved();
        case "down":
        case "j":
          this.move(1);
          return moved();
        case "pgup":
          this.move(-this.listHeight());
          return moved();
        case "pgdown":
          this.move(this.listHeight());
          return moved();
        case "home":
        case "g":
          this.move(-this.rows.length);
          return moved();
        case "end":
        case "G":
          this.move(this.rows.length);
          return moved();
        case "right":
        case "l":
        case "enter":
          return moved(this.open());
        case "left":
        case "h":
          return moved(this.close());
        case "tab":
          this.toggleView();
          return { did: "view" };
        case "s":
          this.sort = (this.sort + 1) % SORT_LABELS.length;
          this.rebuild();
          return { did: "sort" };
        case "m":
          this.editing = true;
          this.input = this.min > 0 ? this.minText : "";
          return { did: "edit" };
        case "c": {
          const r = this.selected();
          if (!r) return {};
          const text = quote(this.path(this.target(r)));
          return { copy: text, effect: this.flash("Copied: " + text), did: "copy" };
        }
        case "r":
          return { action: "rescan", dir: this.rescanTarget() };
        case "?":
          this.help = true;
          return { did: "help" };
      }
      return {};
    }

    header() {
      const view = this.view === "drill" ? "drill-down" : "tree";
      const minLabel = this.min > 0 ? "min " + this.minText : "min off";
      const labels = " · " + SORT_LABELS[this.sort] + " · " + minLabel + " · " + view;
      const title = "dirstato  ";
      const room = this.width - title.length - labels.length;
      const lines = [{ segs: truncate([seg("dirstato", "b"), seg("  " + fitLeft(this.meta.path, room) + labels)], this.width) }];
      if (this.view === "drill") {
        lines.push({ segs: truncate([seg("▸ " + fitLeft(this.path(this.cwd), this.width - 2), "d")], this.width) });
      }
      return lines;
    }

    renderRow(r, styled) {
      const showDate = this.width >= DROP_DATE_BELOW;
      const showCount = this.width >= DROP_COUNT_BELOW;
      let size, files, newest;
      if (r.node) {
        size = r.node.s;
        files = r.node.f;
        newest = r.node.d || "";
      } else {
        size = r.hiddenSize;
        files = r.hiddenFiles;
        newest = "";
      }
      let pct = 100;
      if (r.parent) pct = r.parent.s > 0 ? (100 * size) / r.parent.s : 0;
      const blocks = bar(pct);
      let rest = " ".repeat(BAR_WIDTH - blocks.length);
      if (showCount) rest += " " + padStart(formatCount(files), 6);
      if (showDate) rest += " " + padStart(newest, 10);
      const indent = this.view === "tree" ? "  ".repeat(r.depth) : "";
      const segs = [
        seg(padStart(formatSize(size), 7) + " " + padStart(String(roundHalfEven(pct)), 3) + "% "),
        seg(blocks, "bar"),
        seg(rest + " " + indent),
      ];
      if (!r.node) {
        segs.push(seg("  … " + formatCount(r.hiddenCount) + " small " + plural(r.hiddenCount, "item"), styled ? "d" : ""));
      } else if (this.isDir(r.node)) {
        const marker = this.view === "tree" && this.expanded.has(r.node) ? "▾ " : "▸ ";
        const label = r.node.p ? r.node.n + "/" : this.meta.path;
        segs.push(seg(marker + label));
      } else {
        segs.push(seg("  " + r.node.n));
      }
      return truncate(segs, this.width);
    }

    legend() {
      if (this.height >= FULL_LEGEND_MIN_HEIGHT) return FULL_LEGEND.map((l) => ({ segs: [seg(l)], legend: true }));
      return [{ segs: truncate([seg(SHORT_LEGEND)], this.width), legend: true }];
    }

    statusLine() {
      if (this.editing) {
        const value = this.input === "" ? [seg("", "caret"), seg("500M, 1.5G, 0", "d")] : [seg(this.input), seg("", "caret")];
        return { segs: [seg("Minimum size: ")].concat(value, [seg("   ⏎ apply · esc cancel", "d")]) };
      }
      if (this.state === "rescanning") {
        const p = this.progress;
        const cur = truncate([seg(p.current)], Math.max(10, this.width - 50));
        return {
          segs: truncate(
            [seg("rescanning… " + formatCount(p.files) + " " + plural(p.files, "file") + " · " + formatSize(p.bytes) + " · ")].concat(cur),
            this.width
          ),
        };
      }
      return { segs: truncate([seg(this.status)], this.width) };
    }

    rule() {
      return { segs: [seg("─".repeat(this.width), "rule")] };
    }

    render() {
      if (this.state === "scanning") return this.renderProgress();
      if (this.state === "launch") return this.fit([{ segs: [seg("$ " + this.command), seg("", "caret")] }]);
      if (this.state === "quit") return this.fit([{ segs: [seg("$ " + this.command)] }, { segs: [seg("$ "), seg("", "caret")] }]);
      if (this.help) {
        let lines = [{ segs: [seg("dirstato · help", "b")] }].concat(HELP_TEXT.map((l) => ({ segs: [seg(l)] })));
        if (lines.length > this.height) lines = lines.slice(0, Math.max(0, this.height - 1)).concat([{ segs: [seg(HELP_CLOSE)] }]);
        return this.fit(lines);
      }
      const lines = this.header().concat([this.rule()]);
      const h = this.listHeight();
      const end = Math.min(this.offset + h, this.rows.length);
      for (let i = this.offset; i < end; i++) {
        if (i === this.cursor) {
          const segs = this.renderRow(this.rows[i], false);
          const pad = Math.max(0, this.width - segWidth(segs));
          lines.push({ segs: segs.concat([seg(" ".repeat(pad))]), selected: true, row: i });
          continue;
        }
        lines.push({ segs: this.renderRow(this.rows[i], true), row: i });
      }
      let filled = end - this.offset;
      if (this.rows.length === 0) {
        lines.push({ segs: [seg("  (empty)", "d")] });
        filled++;
      }
      for (let i = filled; i < h; i++) lines.push({ segs: [] });
      lines.push(this.rule());
      return lines.concat(this.legend(), [this.statusLine()]);
    }

    renderProgress() {
      const p = this.progress;
      return this.fit([
        { segs: [seg("dirstato", "b"), seg("  " + this.meta.path)] },
        { segs: [] },
        {
          segs: [
            seg(
              "scanning… " + formatCount(p.files) + " " + plural(p.files, "file") + " · " + formatCount(p.dirs) + " " + plural(p.dirs, "folder") + " · " + formatSize(p.bytes) + " · " + p.elapsed
            ),
          ],
        },
        { segs: [seg(p.current, "d")] },
        { segs: [] },
        { segs: [seg("q / ctrl+c: cancel", "d")] },
      ]);
    }

    fit(lines) {
      const out = lines.slice(0, this.height).map((l) => Object.assign({}, l, { segs: truncate(l.segs, this.width) }));
      while (out.length < this.height) out.push({ segs: [] });
      return out;
    }
  }

  g.DirstatoTUI = {
    DROP_DATE_BELOW: DROP_DATE_BELOW,
    DROP_COUNT_BELOW: DROP_COUNT_BELOW,
    Model: Model,
    formatSize: formatSize,
    formatCount: formatCount,
    parseSize: parseSize,
    plural: plural,
    bar: bar,
    roundHalfEven: roundHalfEven,
    SORT_LABELS: SORT_LABELS,
  };
})(globalThis);
