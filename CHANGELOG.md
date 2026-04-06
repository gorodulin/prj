# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

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

[0.3.0]: https://github.com/gorodulin/prj/releases/tag/v0.3.0
[0.2.0]: https://github.com/gorodulin/prj/releases/tag/v0.2.0
[0.1.0]: https://github.com/gorodulin/prj/releases/tag/v0.1.0
