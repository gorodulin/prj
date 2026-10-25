package config

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/gorodulin/prj/internal/text"
)

// Field describes a single config field for programmatic access.
type Field struct {
	Key     string                      // JSON tag name, e.g. "projects_folder"
	Hint    string                      // example value for error messages
	Default string                      // runtime default, empty if none
	Get     func(*Config) string        // returns string representation
	Set     func(*Config, string) error // parses and sets

	// EncodeJSON converts a non-empty CLI value to its JSON representation
	// for SetField. Nil means the value is stored as a JSON string.
	EncodeJSON func(value string) (json.RawMessage, error)
}

// Fields lists all config fields in declaration order.
var Fields = []Field{
	{
		Key: "projects_folder", Hint: "/path/to/projects",
		Get: func(c *Config) string { return c.ProjectsFolder },
		Set: func(c *Config, v string) error { c.ProjectsFolder = v; return nil },
	},
	{
		Key: "metadata_folder", Hint: "/path/to/metadata",
		Get: func(c *Config) string { return c.MetadataFolder },
		Set: func(c *Config, v string) error { c.MetadataFolder = v; return nil },
	},
	{
		Key: "links_folder", Hint: "/path/to/links",
		Get: func(c *Config) string { return c.LinksFolder },
		Set: func(c *Config, v string) error { c.LinksFolder = v; return nil },
	},
	{
		// Keep Default in sync with config.DefaultLinkTitleFormat.
		Key: "link_title_format", Hint: "{{.Title}}", Default: "{{.Title}}",
		Get: func(c *Config) string { return c.LinkTitleFormat },
		Set: func(c *Config, v string) error { c.LinkTitleFormat = v; return nil },
	},
	{
		// Keep Default in sync with format.DefaultFormat.
		Key: "list_format", Hint: "json", Default: `{{.ID | red}} {{.Local | flag}} {{.ID | date "YY-MM-DD" | cyan}} {{.Title | yellow}} {{.Tags | join ", " | green}}`,
		Get: func(c *Config) string { return c.ListFormat },
		Set: func(c *Config, v string) error { c.ListFormat = v; return nil },
	},
	{
		// Keep Default in sync with config.LinkKindSymlink used in cmd/link.go.
		Key: "link_kind", Hint: "symlink", Default: "symlink",
		Get: func(c *Config) string { return c.LinkKind },
		Set: func(c *Config, v string) error { c.LinkKind = v; return nil },
	},
	{
		Key: "link_sink_name", Hint: "unsorted",
		Get: func(c *Config) string { return c.LinkSinkName },
		Set: func(c *Config, v string) error { c.LinkSinkName = v; return nil },
	},
	{
		Key: "link_comment_format", Hint: "{{.Tags}}",
		Get: func(c *Config) string { return c.LinkCommentFormat },
		Set: func(c *Config, v string) error { c.LinkCommentFormat = v; return nil },
	},
	{
		// Keep Default in sync with config.DefaultProjectIDType.
		Key: "project_id_type", Hint: "ULID", Default: "ULID",
		Get: func(c *Config) string { return c.ProjectIDType },
		Set: func(c *Config, v string) error { c.ProjectIDType = v; return nil },
	},
	{
		// Keep Default in sync with config.DefaultProjectIDPrefix.
		Key: "project_id_prefix", Hint: "prj", Default: "prj",
		Get: func(c *Config) string { return c.ProjectIDPrefix },
		Set: func(c *Config, v string) error { c.ProjectIDPrefix = v; return nil },
	},
	{
		Key: "machine_name", Hint: "my-machine",
		Get: func(c *Config) string { return c.MachineName },
		Set: func(c *Config, v string) error { c.MachineName = v; return nil },
	},
	{
		Key: "machine_id", Hint: "unique-id",
		Get: func(c *Config) string { return c.MachineID },
		Set: func(c *Config, v string) error { c.MachineID = v; return nil },
	},
	{
		Key: "retention_days", Hint: "90",
		Get: func(c *Config) string { return strconv.Itoa(c.RetentionDays) },
		Set: func(c *Config, v string) error {
			n, err := strconv.Atoi(v)
			if err != nil {
				return fmt.Errorf("retention_days must be an integer (got %q)", v)
			}
			c.RetentionDays = n
			return nil
		},
		EncodeJSON: func(v string) (json.RawMessage, error) {
			n, err := strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("retention_days must be an integer (got %q)", v)
			}
			return json.RawMessage(strconv.Itoa(n)), nil
		},
	},
	{
		// Keep Default in sync with config.DefaultColor.
		Key: "color", Hint: "auto|always|never", Default: "auto",
		Get: func(c *Config) string { return c.Color },
		Set: func(c *Config, v string) error { c.Color = v; return nil },
	},
	{
		// Comma-separated on the CLI, a JSON array in the file.
		Key: "hide_tags", Hint: "archived,done",
		Get: func(c *Config) string { return strings.Join(c.HideTags, ",") },
		Set: func(c *Config, v string) error { c.HideTags = text.ParseTags(v); return nil },
		EncodeJSON: func(v string) (json.RawMessage, error) {
			tags := text.ParseTags(v)
			if tags == nil {
				tags = []string{} // e.g. " , ": store [] rather than null
			}
			return json.Marshal(tags)
		},
	},
}

// FieldByKey returns the field with the given JSON key name.
func FieldByKey(key string) (Field, bool) {
	for _, f := range Fields {
		if f.Key == key {
			return f, true
		}
	}
	return Field{}, false
}

// FieldKeys returns all valid key names in declaration order.
func FieldKeys() []string {
	keys := make([]string, len(Fields))
	for i, f := range Fields {
		keys[i] = f.Key
	}
	return keys
}

// encodeFieldValue returns the JSON representation of a CLI value for key,
// using the field's EncodeJSON when it has one. Unknown keys and fields
// without an encoder are stored as JSON strings.
func encodeFieldValue(key, value string) (json.RawMessage, error) {
	if f, ok := FieldByKey(key); ok && f.EncodeJSON != nil {
		return f.EncodeJSON(value)
	}
	return json.Marshal(value)
}

// IsEmpty reports whether the field value is the zero/unset value for the given config.
func (f Field) IsEmpty(c *Config) bool {
	v := f.Get(c)
	if f.Key == "retention_days" {
		return v == "0"
	}
	return v == ""
}

// ValidKeysHelp returns a formatted string listing all valid keys.
func ValidKeysHelp() string {
	return strings.Join(FieldKeys(), ", ")
}
