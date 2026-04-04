package text

import (
	"strings"
	"testing"
)

func TestExtractMarkdownTitle(t *testing.T) {
	tests := []struct {
		name string
		input string
		want string
	}{
		{
			name:  "h1 heading",
			input: "# My Project\n\nSome text.",
			want:  "My Project",
		},
		{
			name:  "h2 heading",
			input: "## My Project\n\nSome text.",
			want:  "My Project",
		},
		{
			name:  "heading with extra whitespace",
			input: "#   Spaced   Out   Title  \n",
			want:  "Spaced Out Title",
		},
		{
			name:  "heading after blank lines",
			input: "\n\n# After Blanks\n",
			want:  "After Blanks",
		},
		{
			name:  "no heading",
			input: "Just some text\nwithout headings\n",
			want:  "",
		},
		{
			name:  "empty input",
			input: "",
			want:  "",
		},
		{
			name:  "heading inside code fence is skipped",
			input: "```\n# Not A Title\n```\n# Real Title\n",
			want:  "Real Title",
		},
		{
			name:  "heading beyond line limit",
			input: "line\nline\nline\nline\nline\nline\nline\nline\nline\nline\nline\nline\nline\nline\nline\nline\nline\nline\nline\nline\n# Too Late\n",
			want:  "",
		},
		{
			name:  "yaml front matter title",
			input: "---\ntitle: Front Matter Title\n---\n# Heading Title\n",
			want:  "Front Matter Title",
		},
		{
			name:  "yaml front matter quoted title",
			input: "---\ntitle: \"Quoted Title\"\n---\n",
			want:  "Quoted Title",
		},
		{
			name:  "yaml front matter single quoted",
			input: "---\ntitle: 'Single Quoted'\n---\n",
			want:  "Single Quoted",
		},
		{
			name:  "yaml front matter without title falls through to heading",
			input: "---\nauthor: Someone\n---\n# Fallback Heading\n",
			want:  "Fallback Heading",
		},
		{
			name:  "yaml end marker with dots",
			input: "---\nauthor: Someone\n...\n# After Dots\n",
			want:  "After Dots",
		},
		{
			name:  "h1 preferred over h2",
			input: "## Secondary\n# Primary\n",
			want:  "Secondary",
		},
		{
			name:  "real project readme",
			input: "# ZFS folder to dataset conversion\n\n> **Keywords**: #zfs #automation\n",
			want:  "ZFS folder to dataset conversion",
		},
		{
			name:  "--- not on first line is not front matter",
			input: "\n---\ntitle: Not Front Matter\n---\n# Real Title\n",
			want:  "Real Title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractMarkdownTitle(strings.NewReader(tt.input))
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
