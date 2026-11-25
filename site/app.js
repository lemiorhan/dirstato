(function () {
  "use strict";

  const T = globalThis.DirstatoTUI;
  const SCAN = globalThis.DIRSTATO_SCAN;
  const screen = document.getElementById("screen");
  if (!T || !SCAN || !screen) return;

  const META = SCAN.meta;
  const HEIGHT = 24;
  const START_MIN = "4M";
  const COMMAND = "dirstato " + META.path + " --min " + START_MIN;
  const TICK_MS = 100;
  const SCAN_MS = 1200;
  const RESCAN_MS = 700;
  const LAUNCH_MS = 450;
  const TOUR_STEP_MS = 1150;
  const TOUR = ["down", "right", "down", "down", "tab", "right", "left", "tab", "s", "s", "s", "left", "left", "up"];
  const BROWSER_KEYS = {
    ArrowUp: "up",
    ArrowDown: "down",
    ArrowLeft: "left",
    ArrowRight: "right",
    Enter: "enter",
    PageUp: "pgup",
    PageDown: "pgdown",
    Home: "home",
    End: "end",
    Escape: "esc",
    Backspace: "backspace",
    Tab: "tab",
  };
  const KEY_LABELS = { up: "↑", down: "↓", left: "←", right: "→", tab: "Tab", enter: "⏎", pgup: "PgUp", pgdown: "PgDn", home: "Home", end: "End" };

  const pre = document.getElementById("screen-text");
  const note = document.getElementById("note");
  const announcer = document.getElementById("announce");
  const keypad = document.getElementById("keypad");
  const minpad = document.getElementById("minpad");
  const statusline = document.getElementById("status");

  let model = null;
  let cols = measureCols();
  let anim = 0;
  let tourTimer = 0;
  let touring = false;
  let interacted = false;
  let lastKey = "";
  let lastDid = "";
  let visible = false;
  const whenShown = [];

  function reducedMotion() {
    return matchMedia("(prefers-reduced-motion: reduce)").matches;
  }

  function measureCols() {
    const probe = document.createElement("span");
    probe.textContent = "0".repeat(100);
    probe.style.cssText = "position:absolute;visibility:hidden;white-space:pre";
    pre.appendChild(probe);
    const ch = probe.getBoundingClientRect().width / 100;
    probe.remove();
    return Math.max(32, Math.floor(pre.clientWidth / ch));
  }

  function esc(s) {
    return s
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/⏎/g, '<span class="ret">⏎</span>');
  }

  function segHTML(x) {
    const t = esc(x.t);
    if (x.s === "b") return "<b>" + t + "</b>";
    if (x.s === "d") return '<span class="d">' + t + "</span>";
    if (x.s === "bar") return '<span class="bar">' + t + "</span>";
    if (x.s === "caret") return '<span class="caret"></span>';
    return t;
  }

  function lineHTML(l) {
    const isRow = l.row !== undefined;
    const cls = "ln" + (l.selected ? " sel" : "") + (isRow ? " row" : "");
    return '<span class="' + cls + '"' + (isRow ? ' data-row="' + l.row + '"' : "") + ">" + l.segs.map(segHTML).join("") + "</span>";
  }

  function draw() {
    pre.innerHTML = model.render().map(lineHTML).join("");
    note.innerHTML = noteHTML(describe());
    keypad.hidden = model.editing;
    minpad.hidden = !model.editing;
    if (model.editing && keypad.contains(document.activeElement)) minpad.querySelector("button").focus();
    if (!model.editing && minpad.contains(document.activeElement)) keypad.querySelector('[data-key="m"]').focus();
  }

  function count(n, word) {
    return n.toLocaleString("en-US") + " " + T.plural(n, word);
  }

  function folderLabel(n) {
    return n.p ? n.n + "/" : META.path.split("/").pop() + "/";
  }

  function share(size, parent) {
    if (!parent) return 100;
    return parent.s > 0 ? (100 * size) / parent.s : 0;
  }

  function legend(size, pct, files, date, dateMeaning) {
    const narrow = model.width < T.DROP_DATE_BELOW;
    const narrower = model.width < T.DROP_COUNT_BELOW;
    return [
      ["size", T.formatSize(size), "disk usage, as du -skx"],
      ["%", T.roundHalfEven(pct) + "%", "share of the parent"],
      ["bar", T.bar(pct) || " ", "one block per tenth"],
      ["count", T.formatCount(files), "files in the subtree" + (narrower ? "; hidden under " + T.DROP_COUNT_BELOW + " columns" : "")],
      ["date", date || "blank", dateMeaning + (narrow ? "; hidden under " + T.DROP_DATE_BELOW + " columns" : "")],
    ];
  }

  function describe() {
    switch (model.state) {
      case "launch":
        return {
          title: "The command behind this demo",
          text: "It runs once a few rows of the screen are in view. Focus the screen and press Enter to run it now.",
        };
      case "scanning":
        return {
          title: "scanning " + META.path,
          text:
            "This scan of " +
            count(META.folders, "folder") +
            " took " +
            META.seconds +
            " s on " +
            META.scanned +
            ". The replay runs slower so you can watch the counters.",
        };
      case "rescanning":
        return {
          title: "rescanning " + model.progress.path,
          text: "r reads the selected folder from disk again and updates every parent total. Nothing changed on disk here, so the numbers stay put.",
        };
      case "quit":
        return { title: "Back at the shell", text: "q quits dirstato. Press Enter or r to run it again." };
    }
    if (model.help) return { title: "dirstato · help", text: "? opens the help screen inside dirstato. Press ? or Esc to close it." };
    if (model.editing) {
      return {
        title: "Minimum size",
        text: "Type a size such as 1M or 1.5G and press Enter, or pick one below. Smaller items fold into one row per folder; 0 turns the filter off. This demo's scan stops at " + model.floorText + ", so that is its floor.",
      };
    }
    const r = model.selected();
    if (!r) return { title: model.path(model.cwd) + "/", text: "Nothing in this folder is above the minimum size." };
    if (!r.node) {
      const pct = share(r.hiddenSize, r.parent);
      const items = count(r.hiddenCount, "item");
      return {
        title: "… " + r.hiddenCount + " small " + T.plural(r.hiddenCount, "item") + " in " + folderLabel(r.parent),
        text: items + " under " + model.minText + " each, " + T.formatSize(r.hiddenSize) + " together, folded into one row. Press m to change the minimum.",
        rows: legend(r.hiddenSize, pct, r.hiddenFiles, "", "not shown for folded items"),
      };
    }
    const n = r.node;
    const pct = share(n.s, n.p);
    const size = T.formatSize(n.s);
    const rows = legend(n.s, pct, n.f, n.d, "newest change inside");
    if (!n.p) {
      return {
        title: META.path,
        text:
          size +
          " is " +
          n.s.toLocaleString("en-US") +
          " bytes, or " +
          META.duKiB.toLocaleString("en-US") +
          " KiB: exactly what `du -skx` reports for this folder.",
        rows: rows,
      };
    }
    const path = model.path(n);
    if (model.isDir(n)) {
      return {
        title: path + "/",
        text: size + " on disk, " + T.roundHalfEven(pct) + "% of " + folderLabel(n.p) + ". " + count(n.f, "file") + " inside; the newest changed on " + n.d + ".",
        rows: rows,
      };
    }
    return {
      title: path,
      text: size + " on disk, " + T.roundHalfEven(pct) + "% of " + folderLabel(n.p) + ". Modified on " + n.d + ".",
      rows: rows,
    };
  }

  function didHTML() {
    if (lastDid === "your-turn") {
      return '<p class="did">That was a tour. Your turn: click a row, or focus the screen and use the keys below.</p>';
    }
    const what = {
      expanded: "opened the folder",
      collapsed: "closed the folder",
      entered: "entered the folder",
      up: "went up one folder",
      view: model.view === "drill" ? "switched to drill-down: one folder at a time" : "switched back to the tree, selection kept",
      sort: "sorted by " + T.SORT_LABELS[model.sort],
      min: "set the minimum to " + model.minText,
      copy: "copied the path to your clipboard",
      "copy-failed": "could not reach your clipboard",
      move: "moved the cursor",
      nothing: "does nothing here",
      "help-close": "closed the help",
    }[lastDid];
    if (!what || !lastKey) return "";
    return '<p class="did">' + (touring ? "tour · " : "") + "<kbd>" + esc(KEY_LABELS[lastKey] || lastKey) + "</kbd> " + esc(what) + "</p>";
  }

  function noteHTML(d) {
    let html = '<p class="note-title">' + esc(d.title).replace(/\//g, "/<wbr>") + "</p><p>" + esc(d.text).replace(/`([^`]+)`/g, "<code>$1</code>").replace(/\d{4}-\d\d-\d\d/g, '<span class="nw">$&</span>') + "</p>";
    if (d.rows) {
      html += '<dl class="legend">';
      for (const [name, value, meaning] of d.rows) {
        html += "<dt>" + name + '</dt><dd class="val">' + (name === "bar" ? '<span class="bar">' + esc(value) + "</span>" : esc(value)) + "</dd><dd>" + esc(meaning) + "</dd>";
      }
      html += "</dl>";
    }
    return html + didHTML();
  }

  function say() {
    const d = describe();
    announcer.textContent = d.title + ". " + d.text.replace(/`/g, "");
  }

  function dirPaths(node) {
    const out = [];
    (function walk(n) {
      if (!model.isDir(n)) return;
      out.push(model.path(n));
      for (const c of n.c || []) walk(c);
    })(node);
    return out;
  }

  function animate(ms, onFrame, onDone) {
    if (reducedMotion()) {
      onFrame(1);
      onDone();
      return;
    }
    const start = performance.now();
    (function frame() {
      const t = Math.min(1, (performance.now() - start) / ms);
      onFrame(t);
      if (t >= 1) {
        onDone();
        return;
      }
      draw();
      anim = setTimeout(frame, TICK_MS);
    })();
  }

  function runScan() {
    clearTimeout(anim);
    model.state = "scanning";
    const paths = dirPaths(model.root);
    const p = model.progress;
    animate(
      SCAN_MS,
      (t) => {
        p.files = Math.round(SCAN.root.f * t);
        p.dirs = Math.round(META.folders * t);
        p.bytes = Math.round(META.bytes * t);
        p.current = paths[Math.min(paths.length - 1, Math.floor(t * paths.length))];
        p.elapsed = "0s";
      },
      () => {
        model.state = "browsing";
        model.rebuild();
        draw();
        startTour();
      }
    );
  }

  function rescan(dir) {
    clearTimeout(anim);
    model.state = "rescanning";
    const paths = dirPaths(dir);
    const p = model.progress;
    p.path = model.path(dir);
    animate(
      RESCAN_MS,
      (t) => {
        p.files = Math.round(dir.f * t);
        p.bytes = Math.round(dir.s * t);
        p.current = paths[Math.min(paths.length - 1, Math.floor(t * paths.length))];
      },
      () => {
        model.state = "browsing";
        model.rebuild();
        draw();
        say();
      }
    );
  }

  function boot(now) {
    clearTimeout(anim);
    stopTour(false);
    model = new T.Model(SCAN, { width: cols, height: HEIGHT, min: T.parseSize(START_MIN), minText: START_MIN, command: COMMAND });
    lastKey = "";
    lastDid = "";
    if (reducedMotion()) {
      draw();
      return;
    }
    model.state = "launch";
    draw();
    const launched = model;
    const go = () => {
      if (model === launched && model.state === "launch") anim = setTimeout(runScan, now ? LAUNCH_MS / 2 : LAUNCH_MS);
    };
    if (now || visible) go();
    else whenShown.push(go);
  }

  function startTour() {
    if (interacted || reducedMotion()) return;
    touring = true;
    let i = 0;
    const step = () => {
      if (!touring) return;
      if (!visible || document.hidden) {
        tourTimer = setTimeout(step, 400);
        return;
      }
      if (i >= TOUR.length) {
        touring = false;
        lastDid = "your-turn";
        draw();
        return;
      }
      const k = TOUR[i++];
      light(k);
      press(k, false);
      tourTimer = setTimeout(step, TOUR_STEP_MS);
    };
    tourTimer = setTimeout(step, TOUR_STEP_MS);
  }

  function stopTour(byVisitor) {
    if (byVisitor) interacted = true;
    if (!touring) return;
    touring = false;
    clearTimeout(tourTimer);
  }

  function light(k) {
    const b = keypad.querySelector('[data-key="' + CSS.escape(k) + '"]');
    if (!b) return;
    b.classList.add("hit");
    setTimeout(() => b.classList.remove("hit"), 260);
  }

  function clearLater(m, effect) {
    setTimeout(() => {
      m.clearStatus(effect.clearStatus);
      if (model === m) draw();
    }, effect.after);
  }

  function copyPath(text, k) {
    const m = model;
    const failed = (e) => {
      if (model !== m) return;
      clearLater(m, m.flash("Copy failed: " + e.message));
      lastKey = k;
      lastDid = "copy-failed";
      draw();
    };
    const write = navigator.clipboard ? navigator.clipboard.writeText(text) : Promise.reject(new Error("the clipboard is not available"));
    write.catch(failed);
  }

  function press(k, byVisitor) {
    const res = model.key(k);
    if (res.action === "run") {
      clearTimeout(anim);
      runScan();
      return;
    }
    if (res.action === "quit") {
      clearTimeout(anim);
      model.state = "quit";
      model.help = false;
      model.editing = false;
      lastKey = "";
      lastDid = "";
      draw();
      if (byVisitor) say();
      return;
    }
    if (res.action === "restart") {
      boot(true);
      return;
    }
    if (res.action === "rescan") {
      rescan(res.dir);
      return;
    }
    if (res.copy) copyPath(res.copy, k);
    if (res.effect) clearLater(model, res.effect);
    lastKey = k;
    lastDid = res.did || "";
    draw();
    if (byVisitor) say();
  }

  screen.addEventListener("keydown", (e) => {
    if (e.metaKey || e.ctrlKey || e.altKey) return;
    if (e.key === "Tab" && e.shiftKey) return;
    const k = BROWSER_KEYS[e.key] || (e.key.length === 1 ? e.key : null);
    if (!k) return;
    if (k === "esc" && !model.editing && !model.help) {
      screen.blur();
      return;
    }
    e.preventDefault();
    stopTour(true);
    press(k, true);
  });

  screen.addEventListener("focus", () => stopTour(true));

  pre.addEventListener("click", (e) => {
    stopTour(true);
    const ln = e.target.closest(".ln[data-row]");
    if (!ln || model.state !== "browsing" || model.help || model.editing) return;
    const i = Number(ln.dataset.row);
    if (i !== model.cursor) {
      model.cursor = i;
      model.scrollToCursor();
      lastKey = "";
      lastDid = "";
      draw();
      say();
      return;
    }
    const r = model.selected();
    const isOpen = r.node && model.view === "tree" && model.isDir(r.node) && model.expanded.has(r.node);
    press(isOpen ? "left" : "right", true);
  });

  keypad.addEventListener("click", (e) => {
    const b = e.target.closest("button[data-key]");
    if (!b) return;
    stopTour(true);
    press(b.dataset.key, true);
  });

  minpad.addEventListener("click", (e) => {
    const b = e.target.closest("button");
    if (!b) return;
    stopTour(true);
    if (b.dataset.min === undefined) {
      press("esc", true);
      return;
    }
    model.input = b.dataset.min;
    press("enter", true);
  });

  let statusTimer = 0;
  function flashStatus(s) {
    statusline.textContent = s;
    statusline.classList.add("on");
    clearTimeout(statusTimer);
    statusTimer = setTimeout(() => statusline.classList.remove("on"), 3000);
  }

  for (const btn of document.querySelectorAll("button[data-copy]")) {
    let reset;
    btn.addEventListener("click", () => {
      const code = document.getElementById(btn.dataset.copy);
      const text = code.textContent;
      const label = btn.querySelector(".label");
      const copied = () => {
        label.textContent = "copied";
        btn.classList.add("done");
        flashStatus("Copied: " + text);
        clearTimeout(reset);
        reset = setTimeout(() => {
          label.textContent = "copy";
          btn.classList.remove("done");
        }, 2000);
      };
      const selectInstead = () => {
        const range = document.createRange();
        range.selectNodeContents(code);
        const sel = getSelection();
        sel.removeAllRanges();
        sel.addRange(range);
        flashStatus("Selected. Press ⌘C to copy.");
      };
      if (navigator.clipboard) navigator.clipboard.writeText(text).then(copied, selectInstead);
      else selectInstead();
    });
  }

  function refit() {
    const next = measureCols();
    if (next === cols) return;
    cols = next;
    model.width = cols;
    model.scrollToCursor();
    draw();
  }

  new ResizeObserver(refit).observe(pre);
  document.fonts.ready.then(refit);

  new IntersectionObserver(
    (entries) => {
      visible = entries[entries.length - 1].isIntersecting;
      if (visible) whenShown.splice(0).forEach((f) => f());
    },
    { rootMargin: "0px 0px -120px 0px" }
  ).observe(screen);

  keypad.hidden = false;
  boot(false);
})();
