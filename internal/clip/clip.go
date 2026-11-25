package clip

import (
	"os/exec"
	"regexp"
	"strings"
)

var unsafe = regexp.MustCompile(`[^A-Za-z0-9_@%+=:,./-]`)

// Quote returns s ready to paste into a POSIX shell as a single word.
func Quote(s string) string {
	if s == "" {
		return "''"
	}
	if !unsafe.MatchString(s) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func Copy(text string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
