package clip

import (
	"os/exec"
	"testing"
)

func TestQuote(t *testing.T) {
	cases := []struct{ in, want string }{
		{"/Users/a/Code/dirstato", "/Users/a/Code/dirstato"},
		{"/Users/a/node_modules/@types/node-1.2,3:x=y+z%", "/Users/a/node_modules/@types/node-1.2,3:x=y+z%"},
		{"/Users/a/Library/Application Support", "'/Users/a/Library/Application Support'"},
		{"/Users/a/it's here", `'/Users/a/it'\''s here'`},
		{"/Users/a/$HOME/`x`", "'/Users/a/$HOME/`x`'"},
		{"/Users/a/back\\slash", `'/Users/a/back\slash'`},
		{"/Users/a/star*?[x]", "'/Users/a/star*?[x]'"},
		{"/Users/a/semi;amp&pipe|", "'/Users/a/semi;amp&pipe|'"},
		{"/Users/a/çalışma ğüş", "'/Users/a/çalışma ğüş'"},
		{"/Users/a/~tilde", "'/Users/a/~tilde'"},
		{"/Users/a/new\nline", "'/Users/a/new\nline'"},
		{"/Users/a/\"dq\"", `'/Users/a/"dq"'`},
		{"/Users/a/!bang#hash", "'/Users/a/!bang#hash'"},
		{"", "''"},
	}
	for _, c := range cases {
		got := Quote(c.in)
		if got != c.want {
			t.Errorf("Quote(%q) = %q, want %q", c.in, got, c.want)
		}
		out, err := exec.Command("/bin/sh", "-c", "printf %s "+got).Output()
		if err != nil {
			t.Errorf("/bin/sh rejected Quote(%q) = %q: %v", c.in, got, err)
			continue
		}
		if string(out) != c.in {
			t.Errorf("/bin/sh read Quote(%q) back as %q", c.in, out)
		}
	}
}
