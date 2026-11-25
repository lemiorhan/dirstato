package units

import "testing"

func TestParseSize(t *testing.T) {
	ok := []struct {
		in   string
		want int64
	}{
		{"0", 0},
		{"512", 512},
		{"100k", 100 << 10},
		{"500M", 500 << 20},
		{"1.5G", 3 << 29},
		{"1.5 GiB", 3 << 29},
		{"2TB", 2 << 40},
		{"8191P", 8191 << 50},
	}
	for _, c := range ok {
		if got, err := ParseSize(c.in); err != nil || got != c.want {
			t.Errorf("ParseSize(%q) = %d, %v; want %d", c.in, got, err, c.want)
		}
	}
	for _, in := range []string{"", "abc", "-1", "1.5X", "8192P"} {
		if got, err := ParseSize(in); err == nil {
			t.Errorf("ParseSize(%q) = %d, want an error", in, got)
		}
	}
}

func TestFormatSize(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0B"},
		{1023, "1023B"},
		{1024, "1.0K"},
		{1048524, "1023.9K"},
		{1048525, "1.0M"}, // 1023.95K would print as "1024.0K"
		{3 << 29, "1.5G"},
	}
	for _, c := range cases {
		if got := FormatSize(c.in); got != c.want {
			t.Errorf("FormatSize(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}
