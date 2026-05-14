package cmd

import (
	"encoding/json"
	"io"
	"os"
)

// Shared error codes for the bulk-IDs commands (`info`, `edit`). Command-
// specific codes (e.g. errCurrentUnsupportedBulk, errTitleUnsupportedBulk)
// live in their respective files.
const (
	errInvalidIDFormat  = "invalid_id_format"
	errUnknownID        = "unknown_id"
	errMetadataIOFailed = "metadata_io_failed"
	errConfigLoadFailed = "config_load_failed"
	errNoFlagsProvided  = "no_flags_provided"
	errFlagsConflict    = "flags_conflict"
)

// TopError is a top-level invocation error in a JSON envelope.
type TopError struct {
	Code   string `json:"code"`
	Reason string `json:"reason"`
}

// IDError is a per-ID failure in a JSON envelope or JSON Lines stream.
type IDError struct {
	ID     string `json:"id"`
	Code   string `json:"code"`
	Reason string `json:"reason"`
}

// Envelope is the uniform JSON shape emitted by bulk-IDs commands in
// --json mode. Results is parameterized so each command can plug in its
// own per-ID success shape.
type Envelope[T any] struct {
	Error   *TopError `json:"error"`
	DryRun  bool      `json:"dry_run,omitempty"`
	Results []T       `json:"results"`
	Errors  []IDError `json:"errors"`
}

// writeJSONEnvelope writes env to stdout as a single JSON document.
func writeJSONEnvelope[T any](env Envelope[T]) error {
	if env.Results == nil {
		env.Results = []T{}
	}
	if env.Errors == nil {
		env.Errors = []IDError{}
	}
	return json.NewEncoder(os.Stdout).Encode(env)
}

// writeJSONLines emits one JSON line per success result followed by one
// JSON line per per-ID error. The error-line shape is
// {"id":"...","error":{"code":"...","reason":"..."}}.
func writeJSONLines[T any](results []T, errs []IDError) error {
	return writeJSONLinesTo(os.Stdout, results, errs)
}

// writeJSONLinesTo is the io.Writer-keyed variant used by tests.
func writeJSONLinesTo[T any](w io.Writer, results []T, errs []IDError) error {
	enc := json.NewEncoder(w)
	for _, r := range results {
		if err := enc.Encode(r); err != nil {
			return err
		}
	}
	type errLine struct {
		ID    string `json:"id"`
		Error struct {
			Code   string `json:"code"`
			Reason string `json:"reason"`
		} `json:"error"`
	}
	for _, e := range errs {
		var line errLine
		line.ID = e.ID
		line.Error.Code = e.Code
		line.Error.Reason = e.Reason
		if err := enc.Encode(line); err != nil {
			return err
		}
	}
	return nil
}

// exitOnPartialFailure mirrors the cobra-bypass idiom: when JSON/JSONL
// output is already on stdout, we exit non-zero ourselves so cobra's RunE
// printer doesn't dump a duplicate human-readable error.
func exitOnPartialFailure(errCount int) {
	if errCount > 0 {
		os.Exit(1)
	}
}

// dedupeStrings returns the input slice with later duplicates removed,
// preserving first-occurrence order. Returns []string{} for empty input.
func dedupeStrings(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}
