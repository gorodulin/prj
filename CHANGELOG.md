# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

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

[0.1.0]: https://github.com/gorodulin/prj/releases/tag/v0.1.0
