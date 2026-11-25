package main

import (
	"bytes"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

const usageLine = "usage: dirstato [path] [--min SIZE]"

func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "dirstato")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	return bin
}

func run(t *testing.T, bin string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	var out, errOut bytes.Buffer
	cmd := exec.Command(bin, args...)
	cmd.Stdout, cmd.Stderr = &out, &errOut
	err := cmd.Run()
	var exit *exec.ExitError
	switch {
	case errors.As(err, &exit):
		code = exit.ExitCode()
	case err != nil:
		t.Fatalf("running %v: %v", args, err)
	}
	return out.String(), errOut.String(), code
}

func TestCLIHelpAndErrors(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	file := filepath.Join(dir, "file")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	const help = usageLine + "\n" +
		"\n" +
		"  path        folder to scan (default: the current folder)\n" +
		"  --min SIZE  hide items smaller than SIZE, e.g. 100M, 1.5G (base 1024)\n"
	stdout, stderr, code := run(t, bin, "--help")
	if code != 0 || stderr != "" || stdout != help {
		t.Errorf("--help: exit %d, stderr %q, stdout:\n%s\nwant:\n%s", code, stderr, stdout, help)
	}

	for _, tc := range []struct {
		args      []string
		code      int
		firstLine string
		usage     bool
	}{
		{[]string{"--min"}, 2, "dirstato: --min needs a value, e.g. --min 100M", true},
		{[]string{"--bogus"}, 2, "dirstato: unknown flag: --bogus", true},
		{[]string{dir, dir}, 2, "dirstato: only one path can be given", true},
		{[]string{"--min", "5X", dir}, 2, `dirstato: invalid size: "5X" (examples: 500M, 1.5G, 0)`, true},
		{[]string{"--min=99999P", dir}, 2, `dirstato: size too large: "99999P"`, true},
		{[]string{file}, 1, "dirstato: " + file + " is not a folder", false},
	} {
		stdout, stderr, code := run(t, bin, tc.args...)
		first, rest, _ := strings.Cut(stderr, "\n")
		if code != tc.code || first != tc.firstLine || stdout != "" {
			t.Errorf("%v: exit %d, stderr %q, stdout %q; want exit %d and first line %q",
				tc.args, code, stderr, stdout, tc.code, tc.firstLine)
		}
		if got := strings.Contains(rest, usageLine); got != tc.usage {
			t.Errorf("%v: usage shown = %v, want %v; stderr %q", tc.args, got, tc.usage, stderr)
		}
	}
}

var (
	turkishLetters = regexp.MustCompile(`[çğıöşüÇĞİÖŞÜ]`)
	turkishWords   = regexp.MustCompile(`\b(bir|yok|isim|dosya|boyut|iptal|kapat|hareket|tarih|kopyala|tara|kolonlar|bilinmeyen)\b`)
)

func builtGoFiles(t *testing.T) map[string]bool {
	t.Helper()
	out, err := exec.Command("go", "list", "-f", `{{$d := .Dir}}{{range .GoFiles}}{{$d}}/{{.}}{{"\n"}}{{end}}`, "./...").Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]bool{}
	for _, abs := range strings.Fields(string(out)) {
		rel, err := filepath.Rel(wd, abs)
		if err != nil {
			t.Fatal(err)
		}
		files[rel] = true
	}
	return files
}

func TestUserFacingStringsAreEnglish(t *testing.T) {
	parsed := map[string]bool{}
	var sawUsage bool
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") && path != "." {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		parsed[path] = true
		ast.Inspect(f, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			s, err := strconv.Unquote(lit.Value)
			if err != nil {
				t.Errorf("%s: cannot unquote %s", path, lit.Value)
				return true
			}
			sawUsage = sawUsage || strings.HasPrefix(s, usageLine)
			if w := turkishLetters.FindString(s) + turkishWords.FindString(strings.ToLower(s)); w != "" {
				t.Errorf("%s: string %q contains Turkish %q", path, s, w)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	built := builtGoFiles(t)
	for _, f := range []string{"main.go", "internal/ui/render.go", "internal/ui/ui.go"} {
		if !built[f] {
			t.Errorf("go list does not report %s among the built files: %v", f, built)
		}
	}
	for f := range built {
		if !parsed[f] {
			t.Errorf("%s is built but was not scanned", f)
		}
	}
	for f := range parsed {
		if !built[f] {
			t.Errorf("%s was scanned but is not a built file", f)
		}
	}
	if !sawUsage {
		t.Error("the usage text was not among the scanned literals")
	}
}
