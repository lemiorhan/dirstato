"""Checks that the landing page demo (site/tui.js) draws the same screens as dirstato.

Each scenario presses the same keys in dirstato running inside tmux and in the
demo's JavaScript model, then compares the two screens line by line and the
row drawn in reverse video. dirstato scans the folder named in
site/scan-data.js, so that folder must still hold what it held when the demo
data was captured; after it changes, recapture the data with
`go run ./scripts/scan-data <folder> site/scan-data.js`. The number formats
are also checked at rounding ties, against values printed by dirstato's Go
code. Needs go, tmux and node.
"""

import json
import os
import pathlib
import shlex
import shutil
import subprocess
import sys
import tempfile
import time

repo = pathlib.Path(__file__).resolve().parent.parent
site = repo / "site"

scenarios = [
    ("initial", 92, 24, "4M", []),
    ("src-open", 92, 24, "4M", ["j", "Right", "j", "j"]),
    ("drill", 92, 24, "4M", ["j", "Right", "j", "j", "Tab"]),
    ("sort-asc", 92, 24, "4M", ["j", "Right", "s"]),
    ("sort-name", 92, 24, "4M", ["j", "Right", "s", "s"]),
    ("narrow", 60, 24, "4M", ["j", "Right", "j"]),
    ("mid", 80, 24, "4M", ["j", "Right", "G"]),
    ("help", 92, 24, "4M", ["?"]),
    ("help-q", 92, 24, "4M", ["?", "q", "j"]),
    ("tall", 92, 32, "4M", ["j", "Right", "j", "Right"]),
    ("min1M", 92, 24, "4M", ["m", "BSpace", "BSpace", "1", "M", "Enter", "j", "Right"]),
    ("drill-up", 92, 24, "4M", ["j", "Right", "j", "Tab", "Left", "Left"]),
    ("count-off", 69, 24, "4M", ["j", "Right"]),
    ("count-on", 70, 24, "4M", ["j", "Right"]),
    ("date-off", 89, 24, "4M", ["j", "Right"]),
    ("date-on", 90, 24, "4M", ["j", "Right"]),
    ("phone", 42, 24, "4M", []),
    ("phone-src", 42, 24, "4M", ["j", "Right", "j", "j"]),
    ("phone-drill", 42, 24, "4M", ["j", "Right", "j", "Tab"]),
    ("min-w", 32, 24, "4M", ["j", "Right", "j"]),
    ("min-w-help", 32, 24, "4M", ["?"]),
    ("pgdown", 92, 24, "4M", ["j", "Right", "j", "Right", "NPage", "NPage"]),
    ("summary", 92, 24, "4M", ["j", "Right", "G", "k", "k", "k", "k", "k", "k", "Right", "Left"]),
    ("root-tab", 92, 24, "4M", ["Tab"]),
    ("root-tab-left", 92, 24, "4M", ["Tab", "Left", "Tab"]),
    ("sum-open", 92, 24, "4M", ["G", "Right"]),
    ("sum-close", 92, 24, "4M", ["G", "Left"]),
    ("sum-tab", 92, 24, "4M", ["G", "Tab"]),
    ("sum-tab-back", 92, 24, "4M", ["G", "Tab", "Tab"]),
    ("sum-drill-enter", 92, 24, "4M", ["G", "Tab", "Enter"]),
    ("sum-nested-tab", 92, 24, "4M", ["j", "Right", "G", "Tab"]),
    ("sum-nested-tab-left", 92, 24, "4M", ["j", "Right", "G", "Tab", "Left"]),
    ("sum-nested-tab-back", 92, 24, "4M", ["j", "Right", "G", "Tab", "Left", "Tab"]),
    ("sum-nested-left", 92, 24, "4M", ["j", "Right", "G", "Left"]),
    ("sum-nested-right", 92, 24, "4M", ["j", "Right", "G", "Right"]),
    ("pg-128K", 92, 24, "128K", ["Right", "NPage"]),
    ("pg2-128K", 92, 24, "128K", ["j", "Right", "NPage", "NPage"]),
    ("pg-back-128K", 92, 24, "128K", ["j", "Right", "NPage", "NPage", "PPage"]),
    ("end-bottom", 92, 24, "128K", ["j", "Right", "G", "G", "End", "NPage", "j"]),
    ("end-then-up", 92, 24, "128K", ["j", "Right", "G", "k", "k", "PPage"]),
    ("home", 92, 24, "128K", ["j", "Right", "G", "g", "Home", "PPage"]),
    ("ties-desc", 92, 24, "128K", ["j", "Right", "G"]),
    ("ties-asc", 92, 24, "128K", ["j", "Right", "s"]),
    ("ties-asc-end", 92, 24, "128K", ["j", "Right", "s", "G"]),
    ("ties-name", 92, 24, "128K", ["j", "Right", "s", "s", "NPage", "NPage"]),
    ("ties-back-desc", 92, 24, "128K", ["j", "Right", "s", "s", "s", "NPage"]),
    ("drill-pg", 92, 24, "128K", ["j", "Tab", "Enter", "NPage", "NPage", "Left"]),
    ("drill-deep", 80, 24, "128K", ["j", "Tab", "Enter", "j", "Enter", "G", "Left", "Left", "Tab"]),
    ("tree-collapse", 92, 24, "128K", ["j", "Right", "j", "Right", "G", "Left", "Left", "Left"]),
    ("narrow-pg", 60, 24, "128K", ["j", "Right", "NPage", "j", "Right"]),
    ("tall-pg", 92, 32, "128K", ["j", "Right", "NPage", "NPage"]),
    ("w40", 40, 24, "128K", ["j", "Right", "j", "Tab"]),
]

go_formats = [
    ("formatCount", 1050, "1.1K"),
    ("formatCount", 1150, "1.1K"),
    ("formatCount", 1250, "1.2K"),
    ("formatCount", 1850, "1.9K"),
    ("formatCount", 1950, "1.9K"),
    ("formatCount", 1050000, "1.1M"),
    ("formatCount", 1150000, "1.1M"),
    ("formatCount", 1850000, "1.9M"),
    ("formatCount", 1950000, "1.9M"),
    ("formatSize", 1280, "1.2K"),
    ("formatSize", 1331, "1.3K"),
    ("formatSize", 3328, "3.2K"),
]

demo_keys = {
    "Right": "right",
    "Left": "left",
    "Tab": "tab",
    "Enter": "enter",
    "BSpace": "backspace",
    "NPage": "pgdown",
    "PPage": "pgup",
    "Home": "home",
    "End": "end",
}


def tmux(*args):
    return subprocess.run(["tmux", *args], capture_output=True, text=True).stdout


def settled(session, timeout=10):
    last = None
    end = time.monotonic() + timeout
    while time.monotonic() < end:
        time.sleep(0.2)
        screen = tmux("capture-pane", "-t", session, "-p")
        if screen.strip() and screen == last:
            return screen
        last = screen
    sys.exit(f"{session}: the screen did not settle within {timeout} s; last screen:\n{last}")


def real(binary, root, name, w, h, minimum, keys):
    session = f"dirstato-demo-{os.getpid()}-{name}"
    command = shlex.join([str(binary), root, "--min", minimum])
    tmux("new-session", "-d", "-s", session, "-x", str(w), "-y", str(h))
    try:
        tmux("set-option", "-t", session, "remain-on-exit", "on")
        tmux("respawn-pane", "-k", "-t", session, command)
        settled(session)
        for k in keys:
            tmux("send-keys", "-t", session, k)
            time.sleep(0.15)
        screen = settled(session)
        styled = tmux("capture-pane", "-t", session, "-p", "-e")
    finally:
        tmux("kill-session", "-t", session)
    reverse = [i for i, line in enumerate(styled.split("\n")[:h]) if "\x1b[7m" in line]
    return [line.rstrip() for line in screen.split("\n")][:h], reverse


def demo(w, h, minimum, keys):
    js = f"""
    require({json.dumps(str(site / "scan-data.js"))});
    require({json.dumps(str(site / "tui.js"))});
    const m = new DirstatoTUI.Model(DIRSTATO_SCAN, {{width: {w}, height: {h}, min: DirstatoTUI.parseSize({json.dumps(minimum)}), minText: {json.dumps(minimum)}}});
    for (const k of {json.dumps([demo_keys.get(k, k) for k in keys])}) m.key(k);
    const lines = m.render();
    console.log(JSON.stringify({{
      text: lines.map(l => l.segs.map(s => s.t).join("").replace(/\\s+$/, "")),
      reverse: lines.flatMap((l, i) => l.selected ? [i] : []),
    }}));
    """
    out = subprocess.run(["node", "-e", js], capture_output=True, text=True)
    if out.returncode:
        sys.exit(out.stderr)
    got = json.loads(out.stdout)
    return got["text"][:h], got["reverse"]


def demo_formats():
    js = f"""
    require({json.dumps(str(site / "tui.js"))});
    console.log(JSON.stringify({json.dumps(go_formats)}.map(([f, n]) => DirstatoTUI[f](n))));
    """
    out = subprocess.run(["node", "-e", js], capture_output=True, text=True, check=True)
    return json.loads(out.stdout)


def scanned_folder():
    js = f"require({json.dumps(str(site / 'scan-data.js'))}); console.log(DIRSTATO_SCAN.meta.path)"
    out = subprocess.run(["node", "-e", js], capture_output=True, text=True, check=True)
    return out.stdout.strip()


def main():
    for tool in ("go", "tmux", "node"):
        if not shutil.which(tool):
            sys.exit(f"{tool} is not on PATH")
    root = scanned_folder()
    if not os.path.isdir(root):
        sys.exit(f"{root} does not exist; the demo data in site/scan-data.js was captured from it. "
                 "Recapture it from another folder with: go run ./scripts/scan-data <folder> site/scan-data.js")

    with tempfile.TemporaryDirectory() as tmp:
        binary = pathlib.Path(tmp) / "dirstato"
        subprocess.run(["go", "build", "-o", str(binary), "."], cwd=repo, check=True)
        failed = 0
        for name, w, h, minimum, keys in scenarios:
            (a, a_rev), (b, b_rev) = real(binary, root, name, w, h, minimum, keys), demo(w, h, minimum, keys)
            a += [""] * (h - len(a))
            b += [""] * (h - len(b))
            diffs = [(i, x, y) for i, (x, y) in enumerate(zip(a, b)) if x != y]
            bad = bool(diffs) or a_rev != b_rev
            print(f"{name:12} {w}x{h} {'DIFF' if bad else 'OK'}")
            for i, x, y in diffs:
                print(f"   line {i:2} dirstato: {x!r}\n           demo:     {y!r}")
            if a_rev != b_rev:
                print(f"   reverse-video lines: dirstato {a_rev}, demo {b_rev}")
            failed += bad

    print(f"{len(scenarios) - failed} of {len(scenarios)} scenarios identical")
    wrong = [(f, n, want, got) for (f, n, want), got in zip(go_formats, demo_formats()) if got != want]
    for f, n, want, got in wrong:
        print(f"{f}({n}): dirstato {want!r}, demo {got!r}")
    print(f"{len(go_formats) - len(wrong)} of {len(go_formats)} number formats identical")
    sys.exit(1 if failed or wrong else 0)


if __name__ == "__main__":
    main()
