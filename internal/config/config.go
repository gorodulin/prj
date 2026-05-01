package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gorodulin/prj/internal/platform"
	"github.com/gorodulin/prj/internal/project"
)

const appName = "prj"
const configFileName = "config.json"

// Link kind constants.
const (
	LinkKindSymlink     = "symlink"
	LinkKindFinderAlias = "finder-alias"
	LinkKindJunction    = "junction"
)

// Default values for optional config fields.
const (
	DefaultMetadataSuffix  = "_meta"
	DefaultLinkTitleFormat = "{{.Title}}"
	DefaultProjectIDType   = "ULID"
	DefaultProjectIDPrefix = "prj"
	DefaultColor           = "auto"
)


// ValidLinkKinds lists recognized values for LinkKind.
var ValidLinkKinds = platform.SupportedLinkTypes()

// ValidProjectIDTypes lists recognized values for ProjectIDType.
var ValidProjectIDTypes = []string{project.FormatAYMDb, project.FormatUUIDv7, project.FormatULID, project.FormatKSUID}

// ValidColorModes lists recognized values for Color.
var ValidColorModes = []string{"auto", "always", "never"}

// machine_id: at most UUID length, alphanumeric plus . _ -.
const maxMachineIDLen = 36

var validMachineID = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)

// Config holds the prj configuration persisted as JSON.
type Config struct {
	ProjectsFolder  string `json:"projects_folder"`
	MetadataFolder  string `json:"metadata_folder,omitempty"`
	MetadataSuffix  string `json:"metadata_folder_suffix,omitempty"`
	LinksFolder     string `json:"links_folder,omitempty"`
	LinkTitleFormat string `json:"link_title_format,omitempty"`
	ListFormat      string `json:"list_format,omitempty"`
	LinkKind          string `json:"link_kind,omitempty"`
	LinkSinkName      string `json:"link_sink_name,omitempty"`
	LinkCommentFormat string `json:"link_comment_format,omitempty"`
	ProjectIDType   string `json:"project_id_type,omitempty"`
	ProjectIDPrefix string `json:"project_id_prefix,omitempty"`
	MachineName     string `json:"machine_name,omitempty"`
	MachineID       string `json:"machine_id,omitempty"`
	RetentionDays   int    `json:"retention_days,omitempty"`
	Color             string `json:"color,omitempty"`

	explicitKeys map[string]bool // keys present in the JSON file (not defaults)
}

// IsExplicit reports whether the given JSON key was present in the config file
// (as opposed to being filled in by a default).
func (c Config) IsExplicit(key string) bool {
	return c.explicitKeys[key]
}

// DefaultPath returns the platform-appropriate config file path.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve config dir: %w", err)
	}
	return filepath.Join(dir, appName, configFileName), nil
}

// Load reads config from the given path. If path is empty, uses DefaultPath
// and tolerates the file not existing (first-run). If path is explicitly
// provided, the file must exist.
func Load(path string) (Config, error) {
	explicit := path != ""
	if !explicit {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return Config{}, err
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if !explicit && errors.Is(err, os.ErrNotExist) {
			return Config{}, nil
		}
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}

	// First pass: determine which keys were explicitly set in the file.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}
	fileKeys := make(map[string]bool, len(raw))
	for k := range raw {
		fileKeys[k] = true
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}
	cfg.explicitKeys = fileKeys

	// Default metadata suffix when metadata folder is configured.
	if cfg.MetadataFolder != "" && cfg.MetadataSuffix == "" {
		cfg.MetadataSuffix = DefaultMetadataSuffix
	}

	// Default project ID type.
	if cfg.ProjectIDType == "" {
		cfg.ProjectIDType = DefaultProjectIDType
	}

	// Default project ID prefix (only relevant for aYYYYMMDDb).
	if cfg.ProjectIDPrefix == "" {
		cfg.ProjectIDPrefix = DefaultProjectIDPrefix
	}

	// Default color mode (auto-detect TTY).
	if cfg.Color == "" {
		cfg.Color = DefaultColor
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

// validate checks config field values for obvious errors.
// It validates enums, path absoluteness, and dangerous path overlaps.
// It does NOT check whether fields are present — that's command-specific.
func (c Config) Validate() error {
	// Enum: link_kind
	if c.LinkKind != "" && !containsStr(ValidLinkKinds, c.LinkKind) {
		return fmt.Errorf("link_kind %q is not recognized (use %s)", c.LinkKind, JoinQuoted(ValidLinkKinds))
	}

	// project_id_prefix: validated by the project package (same rules as the
	// prefix portion of the aYYYYMMDDb pattern).
	if c.ProjectIDPrefix != "" && !project.IsValidPrefix(c.ProjectIDPrefix) {
		return fmt.Errorf("project_id_prefix must be 1-5 lowercase letters (got %q)", c.ProjectIDPrefix)
	}

	// Enum: project_id_type
	if c.ProjectIDType != "" && !containsStr(ValidProjectIDTypes, c.ProjectIDType) {
		return fmt.Errorf("project_id_type %q is not recognized (use %s)", c.ProjectIDType, JoinQuoted(ValidProjectIDTypes))
	}

	// Enum: color
	if c.Color != "" && !containsStr(ValidColorModes, c.Color) {
		return fmt.Errorf("color %q is not recognized (use %s)", c.Color, JoinQuoted(ValidColorModes))
	}

	// machine_id: bounded length, restricted charset.
	if c.MachineID != "" {
		if len(c.MachineID) > maxMachineIDLen {
			return fmt.Errorf("machine_id must be at most %d characters (got %d)", maxMachineIDLen, len(c.MachineID))
		}
		if !validMachineID.MatchString(c.MachineID) {
			return fmt.Errorf("machine_id may only contain letters, digits, dot, underscore, dash (got %q)", c.MachineID)
		}
	}

	// Paths must be absolute if set.
	for _, pf := range []struct{ name, value string }{
		{"projects_folder", c.ProjectsFolder},
		{"metadata_folder", c.MetadataFolder},
		{"links_folder", c.LinksFolder},
	} {
		if pf.value != "" && !filepath.IsAbs(pf.value) {
			return fmt.Errorf("%s must be an absolute path (got %q)", pf.name, pf.value)
		}
	}

	// Dangerous overlaps (only when both paths are set).
	if c.LinksFolder != "" && c.ProjectsFolder != "" {
		if c.LinksFolder == c.ProjectsFolder {
			return fmt.Errorf("links_folder and projects_folder must not be the same path")
		}
		if isInsideDir(c.ProjectsFolder, c.LinksFolder) {
			return fmt.Errorf("projects_folder must not be inside links_folder (project folders would be scanned as link tree)")
		}
	}
	if c.LinksFolder != "" && c.MetadataFolder != "" {
		if c.LinksFolder == c.MetadataFolder {
			return fmt.Errorf("links_folder and metadata_folder must not be the same path")
		}
	}
	if c.MetadataFolder != "" && c.ProjectsFolder != "" {
		if c.MetadataFolder == c.ProjectsFolder {
			return fmt.Errorf("metadata_folder and projects_folder must not be the same path")
		}
	}

	return nil
}

// MetadataDir returns the metadata directory path for a project ID.
func (c Config) MetadataDir(id string) string {
	return filepath.Join(c.MetadataFolder, id+c.MetadataSuffix)
}

// IsValidLinkKind reports whether kind is a recognized link type.
func IsValidLinkKind(kind string) bool {
	return containsStr(ValidLinkKinds, kind)
}

func containsStr(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func JoinQuoted(list []string) string {
	parts := make([]string, len(list))
	for i, v := range list {
		parts[i] = "\"" + v + "\""
	}
	return strings.Join(parts, ", ")
}

// isInsideDir reports whether child is a subdirectory of parent.
func isInsideDir(child, parent string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return rel != "." && !strings.HasPrefix(rel, "..")
}

// SetField reads the config file at path, sets a single key to value,
// validates the resulting config, and writes back. Only the target key is
// modified; defaults injected by Load are not persisted. If value is empty,
// the key is removed from the file.
func SetField(path, key, value string) error {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return err
		}
	}

	// Read existing raw JSON (or start fresh).
	raw := make(map[string]json.RawMessage)
	data, err := os.ReadFile(path)
	if err == nil {
		if err := json.Unmarshal(data, &raw); err != nil {
			return fmt.Errorf("parse config %s: %w", path, err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read config %s: %w", path, err)
	}

	// Set or remove the key in the raw map.
	if value == "" {
		delete(raw, key)
	} else {
		encoded, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("encode value: %w", err)
		}
		raw[key] = encoded
	}

	// Special handling: retention_days is an integer in JSON.
	if key == "retention_days" && value != "" {
		raw[key] = json.RawMessage(value)
	}

	// Unmarshal into Config to validate.
	merged, err := json.Marshal(raw)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(merged, &cfg); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Write back ordered JSON from the raw map.
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir %s: %w", dir, err)
	}
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, out, 0644); err != nil {
		return fmt.Errorf("write config %s: %w", path, err)
	}
	return nil
}

// Save writes config to the given path. If path is empty, uses DefaultPath.
// Creates the parent directory if it does not exist.
func Save(cfg Config, path string) error {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return err
		}
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write config %s: %w", path, err)
	}

	return nil
}
