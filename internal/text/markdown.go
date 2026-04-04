package text

import (
	"bufio"
	"io"
	"strings"
)

const maxScanLines = 20

// ExtractMarkdownTitle extracts the title from markdown content.
//
// Priority:
//  1. YAML front matter "title" field (if present)
//  2. First # or ## heading
//
// Scans at most 20 lines. Skips headings inside fenced code blocks.
// Returns empty string if no title is found.
func ExtractMarkdownTitle(r io.Reader) string {
	scanner := bufio.NewScanner(r)
	line := 0
	inFrontMatter := false
	frontMatterDone := false
	inCodeFence := false

	for scanner.Scan() {
		if line >= maxScanLines {
			break
		}
		raw := scanner.Text()
		trimmed := strings.TrimSpace(raw)
		line++

		// YAML front matter: starts with --- on line 1
		if line == 1 && trimmed == "---" {
			inFrontMatter = true
			continue
		}

		if inFrontMatter {
			if trimmed == "---" || trimmed == "..." {
				inFrontMatter = false
				frontMatterDone = true
				continue
			}
			if title := parseFrontMatterTitle(trimmed); title != "" {
				return title
			}
			continue
		}

		// Skip fenced code blocks
		if strings.HasPrefix(trimmed, "```") {
			inCodeFence = !inCodeFence
			continue
		}
		if inCodeFence {
			continue
		}

		// Match # or ## heading
		if strings.HasPrefix(raw, "# ") {
			return normalizeHeading(raw[2:])
		}
		if strings.HasPrefix(raw, "## ") {
			return normalizeHeading(raw[3:])
		}

		_ = frontMatterDone
	}

	return ""
}

// parseFrontMatterTitle extracts value from a "title: ..." line.
func parseFrontMatterTitle(line string) string {
	if !strings.HasPrefix(line, "title:") && !strings.HasPrefix(line, "Title:") {
		return ""
	}
	val := line[6:] // len("title:") == 6
	val = strings.TrimSpace(val)
	// Remove surrounding quotes if present
	if len(val) >= 2 && (val[0] == '"' || val[0] == '\'') && val[len(val)-1] == val[0] {
		val = val[1 : len(val)-1]
	}
	return strings.TrimSpace(val)
}

// normalizeHeading collapses whitespace and trims a heading string.
func normalizeHeading(s string) string {
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}
