package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/lemiorhan/dirstato/internal/clip"
	"github.com/lemiorhan/dirstato/internal/ui"
	"github.com/lemiorhan/dirstato/internal/units"
)

const usage = `usage: dirstato [path] [--min SIZE]

  path        folder to scan (default: the current folder)
  --min SIZE  hide items smaller than SIZE, e.g. 100M, 1.5G (base 1024)
`

var errHelp = errors.New("help")

func parseArgs(args []string) (path string, minSize int64, minText string, err error) {
	path = "."
	pathSet := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-h" || a == "--help":
			return "", 0, "", errHelp
		case a == "--min" || strings.HasPrefix(a, "--min="):
			v, ok := strings.CutPrefix(a, "--min=")
			if !ok {
				if i+1 >= len(args) {
					return "", 0, "", errors.New("--min needs a value, e.g. --min 100M")
				}
				i++
				v = args[i]
			}
			if minSize, err = units.ParseSize(v); err != nil {
				return "", 0, "", err
			}
			minText = strings.TrimSpace(v)
		case strings.HasPrefix(a, "-") && a != "-":
			return "", 0, "", fmt.Errorf("unknown flag: %s", a)
		default:
			if pathSet {
				return "", 0, "", errors.New("only one path can be given")
			}
			path, pathSet = a, true
		}
	}
	return path, minSize, minText, nil
}

func main() {
	path, minSize, minText, err := parseArgs(os.Args[1:])
	if errors.Is(err, errHelp) {
		fmt.Print(usage)
		return
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "dirstato: %v\n\n%s", err, usage)
		os.Exit(2)
	}
	if st, err := os.Stat(path); err != nil {
		fmt.Fprintf(os.Stderr, "dirstato: %v\n", err)
		os.Exit(1)
	} else if !st.IsDir() {
		fmt.Fprintf(os.Stderr, "dirstato: %s is not a folder\n", path)
		os.Exit(1)
	}

	final, err := tea.NewProgram(ui.New(ui.Config{Root: path, Min: minSize, MinText: minText, Copy: clip.Copy})).Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "dirstato: %v\n", err)
		os.Exit(1)
	}
	if m, ok := final.(ui.Model); ok && m.Err() != nil {
		fmt.Fprintf(os.Stderr, "dirstato: %v\n", m.Err())
		os.Exit(1)
	}
}
