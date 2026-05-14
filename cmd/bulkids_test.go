package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestEnvelope_RoundTrip_Empty(t *testing.T) {
	env := Envelope[int]{
		Results: []int{},
		Errors:  []IDError{},
	}
	got, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `{"error":null,"results":[],"errors":[]}`
	if string(got) != want {
		t.Errorf("marshal = %s, want %s", got, want)
	}
}

func TestEnvelope_DryRunTrue(t *testing.T) {
	env := Envelope[int]{
		DryRun:  true,
		Results: []int{},
		Errors:  []IDError{},
	}
	got, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `{"error":null,"dry_run":true,"results":[],"errors":[]}`
	if string(got) != want {
		t.Errorf("marshal = %s, want %s", got, want)
	}
}

func TestEnvelope_DryRunFalseOmitted(t *testing.T) {
	env := Envelope[int]{
		DryRun:  false,
		Results: []int{},
		Errors:  []IDError{},
	}
	got, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `{"error":null,"results":[],"errors":[]}`
	if string(got) != want {
		t.Errorf("marshal = %s, want %s (dry_run must be omitted)", got, want)
	}
}

func TestEnvelope_TopError(t *testing.T) {
	env := Envelope[int]{
		Error:   &TopError{Code: errNoFlagsProvided, Reason: "nothing to edit"},
		Results: []int{},
		Errors:  []IDError{},
	}
	b, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Envelope[int]
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Error == nil || got.Error.Code != errNoFlagsProvided {
		t.Errorf("top error not preserved: %+v", got.Error)
	}
}

func TestDedupeStrings(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{name: "empty", in: nil, want: []string{}},
		{name: "empty slice", in: []string{}, want: []string{}},
		{name: "single", in: []string{"x"}, want: []string{"x"}},
		{name: "no dupes", in: []string{"a", "b", "c"}, want: []string{"a", "b", "c"}},
		{name: "preserves first occurrence", in: []string{"a", "b", "a", "c", "b"}, want: []string{"a", "b", "c"}},
		{name: "all same", in: []string{"a", "a", "a"}, want: []string{"a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dedupeStrings(tt.in)
			if !sliceEqual(got, tt.want) {
				t.Errorf("dedupeStrings(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

type jsonlSample struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

func TestWriteJSONLines_OnlyResults(t *testing.T) {
	var buf bytes.Buffer
	results := []jsonlSample{
		{ID: "a", Title: "Alpha"},
		{ID: "b", Title: "Beta"},
	}
	if err := writeJSONLinesTo(&buf, results, nil); err != nil {
		t.Fatalf("writeJSONLinesTo: %v", err)
	}
	lines := splitNonEmpty(buf.String())
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2: %q", len(lines), buf.String())
	}
	var first jsonlSample
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("unmarshal line 1: %v", err)
	}
	if first.ID != "a" {
		t.Errorf("line 1 id = %q, want a", first.ID)
	}
}

func TestWriteJSONLines_MixedResultsAndErrors(t *testing.T) {
	var buf bytes.Buffer
	results := []jsonlSample{{ID: "a", Title: "Alpha"}}
	errs := []IDError{{ID: "bogus", Code: errUnknownID, Reason: "unknown project bogus"}}
	if err := writeJSONLinesTo(&buf, results, errs); err != nil {
		t.Fatalf("writeJSONLinesTo: %v", err)
	}
	lines := splitNonEmpty(buf.String())
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2: %q", len(lines), buf.String())
	}

	var ok jsonlSample
	if err := json.Unmarshal([]byte(lines[0]), &ok); err != nil {
		t.Fatalf("unmarshal success line: %v", err)
	}
	if ok.ID != "a" {
		t.Errorf("success line id = %q, want a", ok.ID)
	}

	var errLine struct {
		ID    string `json:"id"`
		Error struct {
			Code   string `json:"code"`
			Reason string `json:"reason"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(lines[1]), &errLine); err != nil {
		t.Fatalf("unmarshal error line: %v", err)
	}
	if errLine.ID != "bogus" {
		t.Errorf("error line id = %q, want bogus", errLine.ID)
	}
	if errLine.Error.Code != errUnknownID {
		t.Errorf("error line code = %q, want %q", errLine.Error.Code, errUnknownID)
	}
}

func TestWriteJSONLines_EmptyInputs(t *testing.T) {
	var buf bytes.Buffer
	if err := writeJSONLinesTo[jsonlSample](&buf, nil, nil); err != nil {
		t.Fatalf("writeJSONLinesTo: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output, got %q", buf.String())
	}
}

func splitNonEmpty(s string) []string {
	raw := strings.Split(strings.TrimRight(s, "\n"), "\n")
	out := make([]string, 0, len(raw))
	for _, l := range raw {
		if l != "" {
			out = append(out, l)
		}
	}
	return out
}
