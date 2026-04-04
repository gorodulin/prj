package text

import (
	"sort"
	"strings"
)

// NormalizeTags deduplicates, lowercases, strips leading # from each tag,
// removes empty entries, and returns them sorted.
func NormalizeTags(tags []string) []string {
	seen := make(map[string]bool, len(tags))
	var result []string

	for _, t := range tags {
		t = strings.TrimPrefix(t, "#")
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		result = append(result, t)
	}

	sort.Strings(result)
	return result
}

// ParseTags splits a comma-separated tag string into a normalized tag slice.
// Returns nil if the input is empty.
func ParseTags(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	var tags []string
	for _, t := range strings.Split(raw, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			tags = append(tags, t)
		}
	}
	return NormalizeTags(tags)
}

// FormatTags returns tags as a single string with # prefixes, space-separated.
func FormatTags(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	parts := make([]string, len(tags))
	for i, t := range tags {
		parts[i] = "#" + t
	}
	return strings.Join(parts, " ")
}
