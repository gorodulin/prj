package platform

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestSymlinkPrivilegeError(t *testing.T) {
	tests := []struct {
		name string
		err  *SymlinkPrivilegeError
		want []string // substrings expected in Error()
	}{
		{
			name: "explicit symlink",
			err:  &SymlinkPrivilegeError{LinkPath: `C:\links\foo`, Target: `C:\projects\01`, FellBackFromJunction: false},
			want: []string{`C:\links\foo`, "Developer Mode"},
		},
		{
			name: "fallback from junction (cross-volume)",
			err:  &SymlinkPrivilegeError{LinkPath: `C:\links\foo`, Target: `D:\projects\01`, FellBackFromJunction: true},
			want: []string{`C:\links\foo`, `D:\projects\01`, "junction not possible across volumes"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.err.Error()
			for _, want := range tt.want {
				if !strings.Contains(msg, want) {
					t.Errorf("Error() = %q, missing %q", msg, want)
				}
			}
		})
	}
}

func TestSymlinkPrivilegeErrorAs(t *testing.T) {
	err := error(&SymlinkPrivilegeError{LinkPath: "/x", Target: "/y"})
	wrapped := fmt.Errorf("create link: %w", err)
	var spe *SymlinkPrivilegeError
	if !errors.As(wrapped, &spe) {
		t.Fatal("errors.As should unwrap SymlinkPrivilegeError")
	}
	if spe.LinkPath != "/x" {
		t.Errorf("LinkPath = %q, want %q", spe.LinkPath, "/x")
	}
}
