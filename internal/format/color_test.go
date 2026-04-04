package format

import (
	"os"
	"testing"
)

func TestIsTTY(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	defer r.Close()
	defer w.Close()

	if IsTTY(r) {
		t.Error("pipe read end should not be a TTY")
	}
	if IsTTY(w) {
		t.Error("pipe write end should not be a TTY")
	}
}

func TestFuncMapColorEnabled(t *testing.T) {
	fm := FuncMap(true)
	tests := []struct {
		name string
		fn   string
		want string
	}{
		{"bold", "bold", "\033[1mhi\033[0m"},
		{"dim", "dim", "\033[2mhi\033[0m"},
		{"red", "red", "\033[31mhi\033[0m"},
		{"green", "green", "\033[32mhi\033[0m"},
		{"yellow", "yellow", "\033[33mhi\033[0m"},
		{"blue", "blue", "\033[34mhi\033[0m"},
		{"cyan", "cyan", "\033[36mhi\033[0m"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := fm[tt.fn].(func(string) string)
			got := fn("hi")
			if got != tt.want {
				t.Errorf("%s(\"hi\") = %q, want %q", tt.fn, got, tt.want)
			}
		})
	}
}

func TestFuncMapColorDisabled(t *testing.T) {
	fm := FuncMap(false)
	for _, name := range []string{"bold", "dim", "red", "green", "yellow", "blue", "cyan"} {
		t.Run(name, func(t *testing.T) {
			fn := fm[name].(func(string) string)
			got := fn("hi")
			if got != "hi" {
				t.Errorf("%s(\"hi\") = %q, want %q", name, got, "hi")
			}
		})
	}
}

func TestUpperLowerFuncs(t *testing.T) {
	fm := FuncMap(false)
	upper := fm["upper"].(func(string) string)
	lower := fm["lower"].(func(string) string)

	tests := []struct {
		name string
		fn   func(string) string
		in   string
		want string
	}{
		{"upper basic", upper, "hello", "HELLO"},
		{"upper already", upper, "HELLO", "HELLO"},
		{"upper empty", upper, "", ""},
		{"lower basic", lower, "HELLO", "hello"},
		{"lower already", lower, "hello", "hello"},
		{"lower empty", lower, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn(tt.in)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDateFunc(t *testing.T) {
	fm := FuncMap(false)
	date := fm["date"].(func(string, string) string)

	tests := []struct {
		name   string
		format string
		id     string
		want   string
	}{
		{"year from aYYYYMMDDb", "YYYY", "p20260402a", "2026"},
		{"short year", "YY", "p20260402a", "26"},
		{"full date", "YYYY-MM-DD", "p20260402a", "2026-04-02"},
		{"day.month.year", "DD.MM.YYYY", "p20260402a", "02.04.2026"},
		{"unrecognized ID", "YYYY", "not-an-id", ""},
		{"empty ID", "YYYY", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := date(tt.format, tt.id)
			if got != tt.want {
				t.Errorf("date(%q, %q) = %q, want %q", tt.format, tt.id, got, tt.want)
			}
		})
	}
}

func TestFlagFunc(t *testing.T) {
	fm := FuncMap(false)
	flag := fm["flag"].(func(bool) string)

	tests := []struct {
		name string
		in   bool
		want string
	}{
		{"true", true, "+"},
		{"false", false, "-"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := flag(tt.in)
			if got != tt.want {
				t.Errorf("flag(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestJoinFunc(t *testing.T) {
	fm := FuncMap(false)
	join := fm["join"].(func(string, []string) string)

	tests := []struct {
		name string
		sep  string
		in   []string
		want string
	}{
		{"basic", ",", []string{"a", "b"}, "a,b"},
		{"single", ",", []string{"a"}, "a"},
		{"empty slice", ",", []string{}, ""},
		{"nil slice", ",", nil, ""},
		{"space sep", " ", []string{"x", "y", "z"}, "x y z"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := join(tt.sep, tt.in)
			if got != tt.want {
				t.Errorf("join(%q, %v) = %q, want %q", tt.sep, tt.in, got, tt.want)
			}
		})
	}
}
