package format

import (
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/gorodulin/prj/internal/project"
)

// IsTTY reports whether f is a terminal (character device).
func IsTTY(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// FuncMap returns template functions for string manipulation and color.
// When color is false, color functions return their input unchanged.
func FuncMap(color bool) template.FuncMap {
	wrap := func(code string) func(string) string {
		if !color {
			return func(s string) string { return s }
		}
		return func(s string) string {
			return fmt.Sprintf("\033[%sm%s\033[0m", code, s)
		}
	}
	// date extracts a timestamp from a project ID and formats it.
	// Token syntax: YYYY, YY, MM, DD, HH, mm, ss.
	// Returns empty string if the ID format is not recognized.
	dateReplacer := strings.NewReplacer(
		"YYYY", "2006",
		"YY", "06",
		"MM", "01",
		"DD", "02",
		"HH", "15",
		"mm", "04",
		"ss", "05",
	)

	return template.FuncMap{
		"join":  func(sep string, elems []string) string { return strings.Join(elems, sep) },
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"date": func(format string, id string) string {
			t, ok := project.ParseIDTime(id)
			if !ok {
				return ""
			}
			return t.Local().Format(dateReplacer.Replace(format))
		},
		"flag": func(b bool) string {
			if b {
				return "+"
			}
			return "-"
		},
		"bold": wrap("1"),
		"dim":    wrap("2"),
		"red":    wrap("31"),
		"green":  wrap("32"),
		"yellow": wrap("33"),
		"blue":   wrap("34"),
		"cyan":   wrap("36"),
	}
}
