package linktree

import (
	"fmt"
	"os"
	"strings"
	"text/template"
)

// ProjectEntry holds the fields needed for link naming.
type ProjectEntry struct {
	ID    string
	Title string
}

// renderLinkName executes a parsed template for a single project entry,
// sanitizes the result for use as a filename, and falls back to the
// project ID if the result is empty.
func renderLinkName(tmpl *template.Template, entry ProjectEntry) string {
	var buf strings.Builder
	if err := tmpl.Execute(&buf, entry); err != nil {
		return entry.ID
	}
	name := strings.TrimSpace(buf.String())
	if name == "" {
		return entry.ID
	}
	name = sanitizeLinkName(name)
	if strings.Trim(name, "- ") == "" {
		return entry.ID
	}
	return name
}

// migrateOldFormat converts legacy {token} syntax to Go template syntax.
// If the format uses old tokens ({title}, {id}, etc.) it is silently
// rewritten. New {{.Field}} syntax passes through unchanged.
var oldFormatWarned bool

func migrateOldFormat(format string) string {
	if strings.Contains(format, "{") && !strings.Contains(format, "{{") {
		if !oldFormatWarned {
			fmt.Fprintf(os.Stderr, "prj: link_title_format uses deprecated {token} syntax; migrate to Go template: replace {title} with {{.Title}}, {id} with {{.ID}}\n")
			oldFormatWarned = true
		}
		return strings.NewReplacer(
			"{title}", "{{.Title}}", "{project_title}", "{{.Title}}",
			"{id}", "{{.ID}}", "{project_id}", "{{.ID}}",
		).Replace(format)
	}
	return format
}

// sanitizeLinkName makes a string safe as a filename on all platforms.
// Applies the strictest union of macOS, Linux, and Windows rules because
// sync tools (like Resilio/Syncthing) may copy these names between machines.
//
// Rules:
//   - Replace forbidden characters: / \ : * ? " < > | and control chars (0x00-0x1F)
//   - Trim trailing dots and spaces (Windows silently strips them)
//   - Mangle Windows reserved device names (CON, PRN, NUL, COM1, etc.)
//   - Truncate to 255 bytes (common limit across ext4, APFS, NTFS)
func sanitizeLinkName(name string) string {
	var b strings.Builder
	b.Grow(len(name))
	for _, r := range name {
		if r < 0x20 || strings.ContainsRune("/\\:*?\"<>|", r) {
			b.WriteByte('-')
		} else {
			b.WriteRune(r)
		}
	}

	result := b.String()

	// Windows forbids trailing dots and spaces.
	result = strings.TrimRight(result, ". ")

	// Windows reserved device names (case-insensitive, with or without extension).
	result = mangleReservedName(result)

	// Filesystem component length limit (255 bytes).
	// Truncate at rune boundary to avoid splitting multi-byte UTF-8 chars.
	if len(result) > 255 {
		result = truncateUTF8(result, 255)
	}

	return result
}

// truncateUTF8 truncates s to at most maxBytes without splitting a multi-byte character.
func truncateUTF8(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	// Walk backward from the limit to find a rune boundary.
	for maxBytes > 0 && maxBytes < len(s) {
		// UTF-8 continuation bytes start with 10xxxxxx.
		if s[maxBytes]&0xC0 != 0x80 {
			break
		}
		maxBytes--
	}
	return s[:maxBytes]
}

// windowsReserved lists device names that cannot be used as filenames on Windows.
var windowsReserved = map[string]bool{
	"CON": true, "PRN": true, "AUX": true, "NUL": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true,
	"COM5": true, "COM6": true, "COM7": true, "COM8": true, "COM9": true,
	"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true,
	"LPT5": true, "LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
}

// mangleReservedName prefixes Windows reserved names with "_" to avoid conflicts.
// "CON" → "_CON", "nul.txt" → "_nul.txt". Non-reserved names pass through.
func mangleReservedName(name string) string {
	// Check the base part before any extension.
	base := name
	if dot := strings.IndexByte(name, '.'); dot >= 0 {
		base = name[:dot]
	}
	if windowsReserved[strings.ToUpper(base)] {
		return "_" + name
	}
	return name
}

// ResolveNames computes final link names for a set of projects landing in
// the same folder. When multiple projects produce the same name and the
// format doesn't already contain the project ID, ALL colliders get an
// " (id)" suffix. Returns projectID → linkName.
func ResolveNames(projects []ProjectEntry, format string, fm template.FuncMap) map[string]string {
	format = migrateOldFormat(format)

	// If the format already embeds the ID, names are inherently unique.
	formatHasID := strings.Contains(format, ".ID")

	tmpl, err := template.New("link").Funcs(fm).Parse(format)
	if err != nil {
		// Bad template — fall back to just IDs.
		result := make(map[string]string, len(projects))
		for _, p := range projects {
			result[p.ID] = p.ID
		}
		return result
	}

	// First pass: compute raw names.
	raw := make(map[string]string, len(projects))
	for _, p := range projects {
		raw[p.ID] = renderLinkName(tmpl, p)
	}

	if formatHasID {
		return raw
	}

	// Detect collisions: count occurrences of each name.
	counts := make(map[string]int, len(raw))
	for _, name := range raw {
		counts[name]++
	}

	// Second pass: disambiguate colliders.
	result := make(map[string]string, len(projects))
	for _, p := range projects {
		name := raw[p.ID]
		if counts[name] > 1 {
			result[p.ID] = name + " (" + p.ID + ")"
		} else {
			result[p.ID] = name
		}
	}

	return result
}
