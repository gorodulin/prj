# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Changed
- `prj find --tags <tags>` works without query terms and shows every
  project, local or metadata-only, that has all the given tags. A bare
  `prj find` is still an error.

### Fixed
- `prj list` and `prj find` sort `aYYYYMMDDb` IDs in creation order when a
  day has more than 26 projects: `p20260402z` now comes before
  `p20260402aa`.

## [0.11.1] - 2026-09-24

### Fixed
- `prj new --title`/`--tags` with `machine_name`, `machine_id`, or
  `metadata_folder` unset no longer leaves an empty project folder behind.
  Required configuration is checked before the project ID is allocated, and
  if a later step fails, the just-created project folder is removed.
- A mistyped or missing `metadata_folder` no longer causes `prj new` and
  `prj edit` to silently create a new directory tree and store metadata
  there. They now fail with `metadata folder <path> does not exist`; the
  folder must exist (`prj init` offers to create it).
- `prj fixrefs --apply` no longer loses a symlink when creating its
  replacement fails; the original link is kept.

## [0.11.0] - 2026-08-25

### Added
- Global `--color=auto|always|never` flag overrides the `color` config field
  per invocation, so `prj list --color=always | tail` keeps color through a
  pipe. `--no-color` still works and takes precedence over `--color`.
- Template color functions now cover the full ANSI standard set: all eight
  base foreground colors (`black`, `red`, `green`, `yellow`, `blue`,
  `magenta`, `cyan`, `white`), their `bright*` variants, plus the `italic`
  and `underline` text styles.

### Changed
- The default `list_format` now colors tags green instead of blue.

## [0.10.0] - 2026-08-21

### Added
- `prj find <query-term>...` searches all known projects — local and
  metadata-only — by case-insensitive substring terms matched against ID,
  title, and tags. Multiple terms combine with OR logic; `--tags` combines
  with AND. Supports the same `-f/--format`, `--json`, and `--jsonl` output
  options as `prj list`. At least one query term is required.
- `prj list` accepts positional query terms: `prj list moto bike` filters
  with OR logic across terms, same matching as `-q`. The `-q` flag remains
  as a compatibility alias and OR-merges with positional terms. Terms that
  start with a dash can be passed after `--`.
- Bare `prj info`, `prj path`, and `prj edit` default to the project
  containing the current working directory (equivalent to passing
  `current`). `cd "$(prj path)"` jumps to the project root from anywhere
  inside it. Because `edit` mutates metadata, a defaulted edit announces
  `using current project <id>` on stderr (suppressed in `--json`/`--jsonl`
  modes).

### Changed
- `prj list` previously ignored positional arguments silently, listing all
  projects; they are now treated as query terms. A stray argument therefore
  filters the output instead of being dropped.

## [0.9.0] - 2026-06-10

### Added
- `prj fixrefs <project-id>...` repairs cross-project symlinks inside project
  folders, rewriting machine-specific targets as relative paths. A symlink is
  fixed when its target contains exactly one valid project ID component (per
  the configured `project_id_type`); the path before the ID is rebuilt
  relative to the link and the suffix after it is preserved. Dry-run by
  default; `--apply` performs the rewrites, `--verbose` lists unchanged links.
  Accepts multiple IDs and the `current` virtual ID, and is interruptible.

## [0.8.0] - 2026-05-14

### Added
- `prj edit --dry-run` flag for previewing metadata changes without
  writing. Output matches a real run except a `DRY RUN — no metadata
  written` banner is printed to stderr. In `--json` mode the envelope
  carries `"dry_run": true`.
- `prj list --json` / `--jsonl` as shorthand for `--format json|jsonl`.
  Mutually exclusive with each other and with `--format`.
- `prj info` now accepts multiple project IDs. Human-mode bulk output
  prints each project's block separated by a blank line; exit code is
  `1` if any ID fails.
- `prj info --jsonl` and `prj edit --jsonl` for line-oriented JSON
  output (one project record per line; per-ID errors emit
  `{"id":"…","error":{"code":"…","reason":"…"}}`). Mutually exclusive
  with `--json`. In JSONL mode dry-run state is signaled only by the
  stderr `DRY RUN` banner.
- Shared `cmd/bulkids.go` plumbing for the JSON envelope and JSON Lines
  output used by both `info` and `edit`.
- `prj list --tags wip,cli` for filtering by tags with the same
  comma-separated syntax used by `prj new` and `prj edit`.
- `--tag` accepted as a deprecated alias for `--tags` on `prj new` and
  `prj edit` (already canonical there as `--tags`), for cross-command
  symmetry.

### Changed (breaking)
- `prj info --json` now emits an envelope
  `{"error":…,"results":[…],"errors":[…]}` rather than a flat project
  object. Single-ID consumers should read `.results[0]`.
- `prj info` error output under `--json` now uses the structured
  envelope (`error.code`, `error.reason`) instead of
  `{"error":"<msg>"}`.
- `prj list --tag` is no longer repeatable. The flag is now a
  deprecated alias for `--tags` and accepts comma-separated values
  (`--tags wip,cli`). Existing scripts using `--tag wip --tag cli` must
  migrate to `--tags wip,cli`. Cobra prints a deprecation warning when
  `--tag` is used.

### Clarified
- `prj edit` applies tag flags (`--tags`, `--add-tags`, `--remove-tags`)
  to every ID when multiple are given. Only `--title` is rejected in
  bulk mode.

## [0.7.0] - 2026-05-01

### Added
- `prj init` interactive setup wizard — walks through machine identity,
  folders, and project ID format, with auto-detection of the dominant ID
  format from existing project folders
- Native folder picker integrated into the wizard: `osascript` on macOS,
  `zenity`/`kdialog` on Linux, `FolderBrowserDialog` on Windows. Type `?`
  at any folder prompt to browse
- `machine_id` validation: max 36 characters, `[a-zA-Z0-9_.-]` only
- `metadata_folder` and `projects_folder` collision check (must not be
  the same path)

### Fixed
- Folder picker no longer hangs or appears on the wrong screen when the
  wizard runs over SSH. A platform-native `guiAvailable()` predicate
  gates the picker — `$DISPLAY`/`$WAYLAND_DISPLAY` on Linux (so `ssh -X`
  keeps working), absence of `SSH_*` env vars on macOS and Windows
- Folder picker falls back to the user's home directory when the
  configured starting path no longer exists, instead of failing silently

## [0.6.0] - 2026-04-26

### Added
- Windows: NTFS directory junctions (`link_kind: junction`), now the default
  on Windows. Junctions need no Developer Mode or admin privileges, so
  `prj link` works out of the box on a fresh Windows install
- `platform.DefaultLinkKind()` selects the OS-appropriate default
  (`junction` on Windows, `symlink` elsewhere) when `link_kind` is unset
- `--kind junction` flag value, validated per platform

### Changed
- Generalized the existing macOS "fall back to symlink when target missing"
  rule into a single `effectiveLinkKind` function that also handles the
  Windows junction cross-volume case (junctions cannot span drives, so
  cross-volume links transparently fall back to symlink)
- Reconciler tracks link kind as a string (`Kind`) instead of a boolean
  (`IsSymlink`), so symlink ↔ junction migration on Windows is detected
  and resolved via the existing `ActionReplace` path
- `platform.ResolveLink` now reports the resolved link kind, eliminating
  the duplicated readlink/alias ladder in `scan.go`

### Fixed
- Windows `ERROR_PRIVILEGE_NOT_HELD` (1314) on symlink creation now
  surfaces as a contextual error that recommends
  `prj config set link_kind junction` (or, when junction was wanted but
  the target is on a different volume, points at the volume mismatch)
  instead of the raw OS error

## [0.5.0] - 2026-04-19

### Added
- Global `--no-color` flag to disable colored output on any command
- `color` config field: `auto` (default), `always`, or `never`

### Fixed
- ANSI escape codes rendered as literal text on Windows console (PowerShell
  and cmd.exe). `IsTTY` now enables `ENABLE_VIRTUAL_TERMINAL_PROCESSING` on
  modern Windows consoles, and correctly reports no-TTY on legacy conhost
  and mintty/MSYS so no codes leak

## [0.4.1] - 2026-04-09

### Added
- Windows ARM64 binary in cross-compiled release assets
- Shell completions auto-installed in `curl | sh` install script

### Fixed
- Validate `link_kind` against platform-supported types (reject unsupported
  kinds early instead of failing at link creation)

### Changed
- Add `-trimpath` to all build targets for reproducible builds

## [0.4.0] - 2026-04-06

### Added
- Universal install script (`scripts/install.sh`) for Linux, macOS, and FreeBSD
  — works in Docker containers (Alpine, slim, etc.) via `curl | sh` or `wget`
- Release process now uploads cross-compiled binaries as GitHub Release assets

### Fixed
- Project ID resolution when cwd is a symlink (e.g. a link tree entry) now
  correctly reports the target project, not the project containing the symlink

## [0.3.0] - 2026-04-05

### Added
- Configurable project ID prefix (`project_id_prefix`) for the `aYYYYMMDDb`
  format — default changed from `p` to `prj`. Supports 1-5 lowercase letters
  with optional `-` or `_` separator (e.g. `prj`, `prj-`, `dev_`)
- All commands now respect the configured prefix when matching project IDs
- Cross-prefix link coexistence: `prj link` auto-resolves name conflicts
  between project links of different prefixes/formats by appending `(ID)`
  suffix, instead of reporting "blocked by existing file"
- `project.IsAnyValidID` — format-agnostic project ID detector used for
  recognizing foreign project links

### Changed
- `IsValidID` now takes a third `prefix` parameter (mandatory for
  `aYYYYMMDDb`, ignored for other formats)
- `GenerateID` now takes a `prefix` parameter
- `Reconcile` accepts `projectsFolder` for foreign link detection
- Default project ID prefix changed from `p` to `prj`

### Improved
- Extracted `resolveTarget` helper from `probeLink` (DRY)

## [0.2.0] - 2026-04-04

### Added
- `prj config set/get/list/path` — view and modify configuration from the CLI
- Shared config requirement helper with remediation hints showing exact
  `prj config set` commands for missing fields
- `config list` shows explicit values, defaults (dimmed), and unset fields
- Field registry mapping JSON keys to config struct fields

### Changed
- Replace "table" list format alias with colorized default template
- `prj edit` and `prj new --title/--tags` now require `metadata_folder`,
  `machine_name`, and `machine_id` (previously only `metadata_folder`)
- `prj list` returns an error when `projects_folder` is missing (was silent)
- Config values in `config list` are quoted and JSON-escapable for copy-paste

### Fixed
- Link name collision when `link_title_format` uses `.ID` through `date`
  function — names that render identically now always get `(<ID>)` suffix

## [0.1.0] - 2026-04-04

### Added
- `prj new` — create project folders with auto-generated IDs
- `prj list` — list projects with filtering by query and tags
- `prj list --all` — include metadata-only projects
- `prj list --missing` — show only metadata-only projects
- `prj edit` — edit project metadata (title, tags)
- `prj edit --force` — create metadata for remote-only projects
- `prj link` — sync tag-driven link tree (symlinks and Finder aliases)
- `prj link --all` — include metadata-only projects (symlink fallback for Finder alias mode)
- `prj path` — print project folder path
- `prj info` — show project details
- Output formats: table, json, jsonl, Go templates
- Metadata system with edit history and cross-machine sync
- Cross-platform support: macOS, Linux, Windows, FreeBSD
- Shell completions for bash, zsh, fish

[0.4.1]: https://github.com/gorodulin/prj/releases/tag/v0.4.1
[0.4.0]: https://github.com/gorodulin/prj/releases/tag/v0.4.0
[0.3.0]: https://github.com/gorodulin/prj/releases/tag/v0.3.0
[0.2.0]: https://github.com/gorodulin/prj/releases/tag/v0.2.0
[0.1.0]: https://github.com/gorodulin/prj/releases/tag/v0.1.0
