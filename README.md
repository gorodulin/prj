---
title: Projector
tags: [cli, cross-platform, focus, golang, links, metadata, programming, project, project-management, projector, sync]
---

# Projector (prj)

Projector is a cross-platform CLI tool for managing project folders, their metadata, and links.

## Install

### Quick install (Linux, macOS, FreeBSD)

```bash
curl -sSfL https://raw.githubusercontent.com/gorodulin/prj/main/scripts/install.sh | sh
```

To install a specific version:

```bash
VERSION=0.3.0 curl -sSfL https://raw.githubusercontent.com/gorodulin/prj/main/scripts/install.sh | sh
```

Works in Docker containers (Alpine, slim, etc.). Installs to `/usr/local/bin` or `~/.local/bin`.
Shell completions for bash, zsh, and fish are installed automatically when
standard completion directories are found. Restart your shell to activate.

### macOS (Homebrew)

```bash
brew tap gorodulin/tap
brew install prj
```

Shell completions for bash, zsh, and fish are installed automatically.
Try `prj lis<Tab>` to verify. If autocompletion doesn't work, see
[Homebrew Shell Completion](https://docs.brew.sh/Shell-Completion).

### Windows (WinGet)

```
winget install gorodulin.prj
```

Requires Windows 10 1809 or later. For manual PowerShell install or
more details, see [docs/windows-distribution.md](docs/windows-distribution.md).

### From source

Requires Go 1.19+:

```bash
go install github.com/gorodulin/prj@latest
```

Or clone and build:

```bash
git clone https://github.com/gorodulin/prj.git
cd prj
make install
```

## Quick start

Run the interactive setup wizard:

```bash
prj init
```

It walks through machine identity, folders, and project ID format, and offers
a native folder picker on macOS, Linux, and Windows. Then create a project:

```bash
prj new --title "My Project"
```

If you'd rather configure manually, set the keys directly:

```bash
prj config set projects_folder /path/to/your/projects
prj config set metadata_folder /path/to/metadata
prj config set machine_name "my-laptop"
prj config set machine_id "$(uuidgen)"
```

Add `links_folder` to organize projects by topic. See below.

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

Default output (one project per line, colorized on TTY):
```
01J5B3GR41... + 26-04-02 prj (Golang) cli, golang
01J5B3H2K8... + 26-04-01 ZFS conversion automation, zfs
```

JSON output includes all fields (`id`, `title`, `path`, `tags`):
```json
[
  {
    "id": "prj20260402a",
    "title": "prj (Golang)",
    "path": "/Users/.../prj20260402a",
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
prj edit prj20260402a --title "New Title"           # set title
prj edit prj20260402a --tags "cli,golang"           # replace all tags
prj edit prj20260402a --add-tags "new-tag"          # add to existing tags
prj edit prj20260402a --remove-tags "old-tag"       # remove specific tags
prj edit prj20260402a --title "" --tags ""          # clear title and tags
prj edit current --add-tags "wip"                 # edit project in cwd
```

`--tags` and `--add-tags`/`--remove-tags` are mutually exclusive.

Output: `<id><TAB><title>` (or just `<id>` if no title).

Register a project that exists on another machine but not on this one:
```bash
prj edit prj20250101a --force --title "Remote Project" --tags "infra"
```

`--force` creates metadata for a project even if its folder is not
present locally. The ID must still be a valid project ID format.

### `prj link [project-id]`

Sync project links in a user-organized folder tree.

```bash
prj link                    # sync all projects
prj link prj20260402a         # sync one project only
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
+ Programming/golang/prj (Golang)       → prj20260402a
- Photos/Old Holiday                    (prj20250101a)
~ Work/cli/prj (Golang)                → prj20260402a (wrong kind: want symlink)

2 created, 1 removed, 1 replaced
```

Only links created by Projector are touched. Regular files and directories
in the link tree are never modified.

See [docs/link-system.md](docs/link-system.md) for the full design.

### `prj path <project-id>`

Print the full path to a project folder.

```bash
prj path prj20260402a            # /Users/.../projects/prj20260402a
prj path current               # path of project in cwd
prj path prj20260402a --strict   # error if folder doesn't exist
```

Default: prints the path for any valid ID, warns on stderr if the folder doesn't exist locally. Useful as a path builder in scripts.

`--strict`: exits with error code 1 if the folder doesn't exist. Use in scripts that need the folder to be present.

```bash
# Path builder (always succeeds for valid IDs):
dir=$(prj path prj20250101a)

# Strict (fails if not synced):
dir=$(prj path prj20250101a --strict) || echo "not synced"
```

Invalid ID format always errors regardless of `--strict`.

## Config

### `prj config`

View and modify configuration from the command line.

```bash
prj config set projects_folder /path/to/projects
prj config set metadata_folder /path/to/metadata
prj config get projects_folder       # print a single value
prj config list                      # show all keys and values
prj config path                      # print config file location
```

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
  "project_id_type": "aYYYYMMDDb",
  "project_id_prefix": "prj",
  "machine_name": "Newton",
  "machine_id": "newton",
  "color": "auto"
}
```

| Field | Description |
|---|---|
| `projects_folder` | **(required)** Root directory containing project folders |
| `metadata_folder` | Root directory for project metadata (titles, tags, edit history). Metadata directories are named `<project-id>_meta` |
| `project_id_type` | ID format for new projects: `ULID` (default), `UUIDv7`, `KSUID`, `aYYYYMMDDb`. See [Choosing a project ID format](#choosing-a-project-id-format) |
| `project_id_prefix` | Prefix for `aYYYYMMDDb` IDs: 1-5 lowercase letters, optionally followed by `-` or `_` (default: `prj`). Ignored by other formats |
| `machine_name` | Human-readable name for this machine (recorded in metadata) |
| `machine_id` | Machine identifier (recorded in metadata). Up to 36 characters, `[a-zA-Z0-9_.-]` only |
| `retention_days` | Automatically delete metadata entries older than N days. 0 = disabled (default) |
| `links_folder` | Root of the folder tree for `prj link` |
| `link_kind` | Link type: `symlink` (default) or `finder-alias` |
| `link_title_format` | Go template for link names. Fields: `{{.Title}}`, `{{.ID}}`. Supports same functions as `list_format` (`date`, `upper`, `lower`, etc.). Old `{token}` syntax is auto-migrated |
| `list_format` | Default output format for `prj list`: `json`, `jsonl`, or a Go template (e.g. `"{{.ID}}\t{{.Title}}"`). Overridden by `--format` flag |
| `link_sink_name` | Fallback folder name for projects that match no tag-based folder (empty = disabled) |
| `color` | Colored output mode: `auto` (default, on when stdout is a TTY), `always`, or `never`. The global `--no-color` flag overrides this |

## Metadata

The metadata folder stores project titles, tags, and their edit history.
It is separate from the projects folder — configure it via
`prj config set metadata_folder /path/to/metadata`.

With metadata configured, you can:
- See titles and tags in `prj list` output
- Edit titles and tags with `prj edit`
- List projects that aren't present on this machine (`prj list --all` or `--missing`)

### Syncing across machines

Projector does not sync files itself. Use a file sync tool (Resilio Sync,
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
| Example | `01J5B3GR41TSV4RRPQD3NGHX42` | `01932c07-a9c3-7b2a-8f1a-6b3c9d4e5f67` | `2E8JwMKbBEgHvAsD9kNLRqpTiS0` | `prj20260402a` |
| Length | 26 | 36 (with dashes) | 27 | 12-16 |
| Characters | `0-9 A-Z` (no I, L, O, U) | `0-9 a-f` + dashes | `0-9 A-Z a-z` | `a-z 0-9 - _` |
| Time precision | Milliseconds | Milliseconds | Seconds | Day |
| Lexicographic sort | Yes | Yes | Yes | Yes |
| Case-sensitive | No | No | **Yes** | No |
| Globally unique | Yes | Yes | Yes | No (needs collision check) |
| Filesystem-safe | All OS | All OS | **Rare risk on case-insensitive FS** | All OS |

### Recommendations

**ULID** (default) — best general-purpose choice. Short, case-insensitive,
globally unique without collision checks. Safe on macOS (APFS/HFS+) and
Windows (NTFS) which are case-insensitive by default. Crockford Base32
avoids ambiguous characters (0/O, 1/I/L).

**aYYYYMMDDb** — the best choice for personal projects. Human-friendly,
date-based, short and readable (`prj20260402a`). You can create folders
by hand and instantly see when a project was started. The prefix is
configurable via `project_id_prefix` (default: `prj`). Proven in practice
over 20 years of personal project management. Not globally unique across
machines — requires scanning existing IDs to avoid collisions.

**UUIDv7** — use if your tooling expects standard UUIDs. Same time precision
as ULID but longer (36 chars with dashes).

**KSUID** — use with caution. Mixed-case Base62 means `2E8Jw` and `2e8jw`
are different IDs but point to the same folder on macOS and Windows.
Only safe on case-sensitive filesystems (Linux ext4, ZFS). Collision risk
is extremely low in practice.

## Development

```bash
make help     # show all targets
make build    # compile
make check    # test + lint
make cover    # HTML coverage report
make cross    # cross-compile all platforms
```
