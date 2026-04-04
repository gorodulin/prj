# prj

Cross-platform CLI tool for managing project folders, their metadata, and links.

## Quick start

1. Install: `make install`
2. Create a config file (see [Config](#config) for the path on your platform):
   ```json
   { "projects_folder": "/path/to/your/projects" }
   ```
3. Create your first project:
   ```bash
   prj new --title "My Project"
   ```

That's the minimum setup. Add `metadata_folder` to track titles and tags,
or `links_folder` to organize projects by topic. See below.

## Commands

### `prj list`

List local projects with titles and tags.

```bash
prj list                    # ID, title, tags (local projects only)
prj list --all              # include metadata-only projects (not present locally)
prj list --missing          # show only metadata-only projects
prj list -f json            # JSON array
prj list -f jsonl           # one JSON object per line
prj list | grep zfs         # search across titles and tags
```

#### Output formats

The `--format` / `-f` flag controls output. It accepts a named format
or a Go [text/template](https://pkg.go.dev/text/template) string:

```bash
prj list                    # default template: ID, title, tags
prj list -f json            # JSON array with all fields
prj list -f jsonl           # one JSON object per line
prj list -f '{{.ID}}'      # custom Go template
```

Default output (tab-separated, one project per line):
```
p20260401a	ZFS folder to dataset conversion	automation,zfs
p20260402a	prj (Golang)	cli,golang
```

JSON output includes all fields (`id`, `title`, `path`, `tags`):
```json
[
  {
    "id": "p20260402a",
    "title": "prj (Golang)",
    "path": "/Users/.../p20260402a",
    "tags": ["cli", "golang"]
  }
]
```

#### Custom templates

Pass a Go template string to `--format`:

```bash
prj list -f '{{.ID}}'                          # just IDs
prj list -f '{{.ID}}\t{{.Title}}'              # ID and title
prj list -f '{{.Tags | join ","}}'             # comma-separated tags
prj list -f '{{.ID | green}}\t{{.Title}}'      # with color (TTY only)
prj list -f '{{if .Title}}{{.Title}}{{end}}'   # titles only, skip blanks
```

Available fields: `.ID`, `.Title`, `.Path`, `.Tags` ([]string).

Template functions:
| Function | Example | Description |
|---|---|---|
| `join` | `{{.Tags \| join ","}}` | Join string slice with separator |
| `date` | `{{.ID \| date "YYYY-MM-DD"}}` | Extract date from project ID. Tokens: `YYYY`, `YY`, `MM`, `DD`, `HH`, `mm`, `ss` |
| `upper`, `lower` | `{{.ID \| upper}}` | Change case |
| `bold`, `dim` | `{{.ID \| bold}}` | Text styling (auto-disabled when piped) |
| `red`, `green`, `yellow`, `blue`, `cyan` | `{{.Title \| green}}` | Color (auto-disabled when piped) |

Escape sequences `\t` and `\n` are interpreted in template strings.

#### Default format in config

Set `list_format` in config to change the default (overridden by `--format`):

```json
{
  "list_format": "{{.ID}}\t{{.Title}}"
}
```

#### Title resolution

Title is resolved from metadata if configured, otherwise from the first
`#` heading in the project's `README.md`, otherwise the project ID alone.

By default, only projects with a local folder are shown. `--all` includes
projects known only through metadata (useful for multi-machine sync).
`--missing` shows only metadata-only projects (not present locally).
`--all` and `--missing` are mutually exclusive.

#### Scripting examples

```bash
# Get all project IDs
prj list -f '{{.ID}}'

# Loop over id + path pairs
prj list -f jsonl | jq -r '[.id, .path] | @tsv' | while IFS=$'\t' read -r id path; do
  echo "backing up $id from $path"
done

# Find projects with a specific tag
prj list -f json | jq -r '.[] | select(.tags | index("golang")) | .id'

# Feed IDs into another command
prj list -f '{{.ID}}' | xargs -I{} prj path {}
```

### `prj new`

Create a new project with an auto-generated ID.

```bash
prj new                                          # folder only
prj new --title "My Project"                     # + title
prj new --title "My Project" --tags "cli,golang" # + title and tags
prj new --title "My Project" --readme            # + README.md
```

Output: `<id><TAB><path>` (tab-separated, for use in scripts).

```bash
# Use in scripts:
id=$(prj new --title "Experiment" | cut -f1)
```

`--readme` creates a `README.md` with YAML front matter:
```markdown
---
title: My Project
tags: [cli, golang]
---

# My Project
```

### `prj edit <project-id>`

Edit project metadata (title and/or tags).

```bash
prj edit p20260402a --title "New Title"           # set title
prj edit p20260402a --tags "cli,golang"           # replace all tags
prj edit p20260402a --add-tags "new-tag"          # add to existing tags
prj edit p20260402a --remove-tags "old-tag"       # remove specific tags
prj edit p20260402a --title "" --tags ""          # clear title and tags
prj edit current --add-tags "wip"                 # edit project in cwd
```

`--tags` and `--add-tags`/`--remove-tags` are mutually exclusive.

Output: `<id><TAB><title>` (or just `<id>` if no title).

Register a project that exists on another machine but not on this one:
```bash
prj edit p20250101a --force --title "Remote Project" --tags "infra"
```

`--force` creates metadata for a project even if its folder is not
present locally. The ID must still be a valid project ID format.

### `prj link [project-id]`

Sync project links in a user-organized folder tree.

```bash
prj link                    # sync all projects
prj link p20260402a         # sync one project only
prj link current            # sync project in cwd
prj link --dry-run          # preview changes
prj link --verbose          # include unchanged links in output
prj link --all              # include metadata-only projects
prj link --kind symlink     # override link type
prj link --warn-unplaced    # list projects with no matching placement
```

You organize the link tree yourself — create folders for topics you care
about. `prj link` then places a link for each project into the folders
that match its tags. Folder names are converted to tags
(e.g. `"Photo & Video"` becomes two tags: `photo` and `video`; names are
split on ` & `, lowercased, and spaces become underscores).

A project can appear in multiple folders if several match. If no folder
matches a project's tags and a fallback folder exists (see `link_sink_name`
in [Config](#config)), the project is placed there instead.

Output shows what changed:
```
+ Programming/golang/prj (Golang)       → p20260402a
- Photos/Old Holiday                    (p20250101a)
~ Work/cli/prj (Golang)                → p20260402a (wrong kind: want symlink)

2 created, 1 removed, 1 replaced
```

Only links created by `prj` are touched. Regular files and directories
in the link tree are never modified.

See [docs/link-system.md](docs/link-system.md) for the full design.

### `prj path <project-id>`

Print the full path to a project folder.

```bash
prj path p20260402a            # /Users/.../projects/p20260402a
prj path current               # path of project in cwd
prj path p20260402a --strict   # error if folder doesn't exist
```

Default: prints the path for any valid ID, warns on stderr if the folder doesn't exist locally. Useful as a path builder in scripts.

`--strict`: exits with error code 1 if the folder doesn't exist. Use in scripts that need the folder to be present.

```bash
# Path builder (always succeeds for valid IDs):
dir=$(prj path p20250101a)

# Strict (fails if not synced):
dir=$(prj path p20250101a --strict) || echo "not synced"
```

Invalid ID format always errors regardless of `--strict`.

## Config

Config file location (auto-detected per platform):
- macOS: `~/Library/Application Support/prj/config.json`
- Linux/BSD: `~/.config/prj/config.json`
- Windows: `%AppData%\prj\config.json`

Only `projects_folder` is required. All other fields are optional and
enable additional features.

```json
{
  "projects_folder": "/path/to/projects",
  "metadata_folder": "/path/to/metadata",
  "list_format": "{{.ID}}\t{{.Title}}",
  "links_folder": "/path/to/links",
  "link_kind": "symlink",
  "link_title_format": "{{.ID}} {{.Title}}",
  "link_sink_name": "unsorted",
  "project_id_type": "ULID",
  "machine_name": "Newton",
  "machine_id": "newton"
}
```

| Field | Description |
|---|---|
| `projects_folder` | **(required)** Root directory containing project folders |
| `metadata_folder` | Root directory for project metadata (titles, tags, edit history) |
| `metadata_folder_suffix` | Suffix appended to project ID to form metadata directory names (default: `_meta`) |
| `project_id_type` | ID format for new projects: `ULID` (default), `UUIDv7`, `KSUID`, `aYYYYMMDDb`. See [Choosing a project ID format](#choosing-a-project-id-format) |
| `machine_name` | Human-readable name for this machine (recorded in metadata) |
| `machine_id` | Machine identifier (recorded in metadata) |
| `retention_days` | Automatically delete metadata entries older than N days. 0 = disabled (default) |
| `links_folder` | Root of the folder tree for `prj link` |
| `link_kind` | Link type: `symlink` (default) or `finder-alias` |
| `link_title_format` | Go template for link names. Fields: `{{.Title}}`, `{{.ID}}`. Supports same functions as `list_format` (`date`, `upper`, `lower`, etc.). Old `{token}` syntax is auto-migrated |
| `list_format` | Default output format for `prj list`: `json`, `jsonl`, or a Go template (e.g. `"{{.ID}}\t{{.Title}}"`). Overridden by `--format` flag |
| `link_sink_name` | Fallback folder name for projects that match no tag-based folder (empty = disabled) |

## Metadata

The metadata folder stores project titles, tags, and their edit history.
It is separate from the projects folder — configure it via `metadata_folder`
in `config.json`.

With metadata configured, you can:
- See titles and tags in `prj list` output
- Edit titles and tags with `prj edit`
- List projects that aren't present on this machine (`prj list --all` or `--missing`)

### Syncing across machines

`prj` does not sync files itself. Use a file sync tool (Resilio Sync,
Syncthing, etc.) to sync `projects_folder` and `metadata_folder` across
machines. Metadata folders from different machines can be merged freely —
just combine their contents into one folder.

See [docs/metadata-system.md](docs/metadata-system.md) for the internal design.

### Automatic cleanup

Metadata files accumulate over time but are tiny (~1KB each). To
automatically delete old entries, set `retention_days` in config:

```json
{ "retention_days": 180 }
```

Cleanup runs after `prj new` and `prj edit`. At least 2 entries per
project are always kept. Disabled by default.

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | Error (invalid args, missing config, folder creation failed, etc.) |

Warnings (e.g. missing project folder in non-strict mode) go to stderr but don't affect the exit code.

## Choosing a project ID format

Every project folder is named by its ID. Once you pick a format, all future
projects use it and existing folders stay as they are — there are no aliases
or migration tools. Choose carefully.

Set the format via `project_id_type` in config. Default: `ULID`.

### Comparison

| | ULID | UUIDv7 | KSUID | aYYYYMMDDb |
|---|---|---|---|---|
| Example | `01J5B3GR41TSV4RRPQD3NGHX42` | `01932c07-a9c3-7b2a-8f1a-6b3c9d4e5f67` | `2E8JwMKbBEgHvAsD9kNLRqpTiS0` | `p20260402a` |
| Length | 26 | 36 (with dashes) | 27 | 11-13 |
| Characters | `0-9 A-Z` (no I, L, O, U) | `0-9 a-f` + dashes | `0-9 A-Z a-z` | `a-z 0-9` |
| Time precision | Milliseconds | Milliseconds | Seconds | Day |
| Lexicographic sort | Yes | Yes | Yes | Yes |
| Case-sensitive | No | No | **Yes** | No |
| Globally unique | Yes | Yes | Yes | No (needs collision check) |
| Filesystem-safe | All OS | All OS | **Risk on case-insensitive FS** | All OS |

### Recommendations

**ULID** (default) — best general-purpose choice. Short, case-insensitive,
globally unique without collision checks. Safe on macOS (APFS/HFS+) and
Windows (NTFS) which are case-insensitive by default. Crockford Base32
avoids ambiguous characters (0/O, 1/I/L).

**UUIDv7** — use if your tooling expects standard UUIDs. Same time precision
as ULID but longer (36 chars with dashes).

**KSUID** — use with caution. Mixed-case Base62 means `2E8Jw` and `2e8jw`
are different IDs but point to the same folder on macOS and Windows.
Only safe on case-sensitive filesystems (Linux ext4, ZFS).

**aYYYYMMDDb** — human-friendly, date-based format for personal projects
where you sometimes create folders by hand. Short and readable
(`p20260402a`), but requires scanning existing IDs to avoid collisions
and is not globally unique across machines.

## Development

```bash
make help     # show all targets
make build    # compile
make check    # test + lint
make cover    # HTML coverage report
make cross    # cross-compile all platforms
```
