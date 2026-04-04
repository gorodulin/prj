package text

import (
	"strings"
	"testing"
)

func TestNormalizeTags(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want string
	}{
		{"basic", []string{"foo", "bar"}, "bar,foo"},
		{"strips hash", []string{"#cli", "#golang"}, "cli,golang"},
		{"lowercases", []string{"Foo", "BAR"}, "bar,foo"},
		{"deduplicates", []string{"foo", "Foo", "#foo"}, "foo"},
		{"removes empty", []string{"", " ", "#"}, ""},
		{"nil input", nil, ""},
		{"mixed", []string{"#Zfs", "server", "#ZFS", " infra "}, "infra,server,zfs"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := strings.Join(NormalizeTags(tt.in), ",")
			if got != tt.want {
				t.Errorf("NormalizeTags = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseTags(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"basic", "cli,golang", "cli,golang"},
		{"with spaces", " cli , golang , test ", "cli,golang,test"},
		{"with hashes", "#cli,#golang", "cli,golang"},
		{"empty string", "", ""},
		{"only commas", ",,", ""},
		{"deduplicates", "foo,Foo,FOO", "foo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := strings.Join(ParseTags(tt.in), ",")
			if got != tt.want {
				t.Errorf("ParseTags(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestFormatTags(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want string
	}{
		{"basic", []string{"cli", "golang"}, "#cli #golang"},
		{"single", []string{"foo"}, "#foo"},
		{"empty", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatTags(tt.in)
			if got != tt.want {
				t.Errorf("FormatTags = %q, want %q", got, tt.want)
			}
		})
	}
}
