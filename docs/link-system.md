# Link System

Reference for the project link placement system.
Update this doc as the Go implementation evolves.

## Purpose

Projects live in flat ID-named folders (`p20260402a/`). Humans need to
browse them by topic. The link system maintains a user-organized folder
tree filled with symlinks (or Finder aliases) pointing to project folders.
Tags drive placement — the algorithm decides where each link belongs.

Goals:
- Projects appear in every semantically relevant branch of the tree
- The tree structure is user-controlled (create/rename/move folders freely)
- `prj link` syncs the tree to match current tags — idempotent, deterministic
- Only links are managed; regular files and folders are never touched

## Command

```
prj link [project-id] [flags]
```

No arguments: sync all projects. One argument: sync that project only
(full reconcile — remove stale + create new).

### Flags

| Flag | Description |
|---|---|
| `--dry-run` | Show what would change, touch nothing |
| `--verbose` | Include unchanged links in output |
| `--kind <type>` | Override `link_kind` from config |
| `--warn-unplaced` | List projects that got no placement (not even sink) |
| `--all`, `-a` | Include metadata-only projects (not present locally) |

### Output

Changes only by default. One line per action:

```
+ Work/cli/prj (Golang)                → p20260402a
+ Work/golang/prj (Golang)             → p20260402a
- Photos/Old Holiday                   (stale)
! Work/cli/prj (Golang)                blocked by regular file

2 created, 1 removed, 1 conflict
```

Exit 0 on success, 1 on error. Conflicts are warnings, not errors.

## Config

```json
{
  "links_folder": "/Users/.../Links",
  "link_kind": "symlink",
  "link_title_format": "{{.Title}}",
  "link_sink_name": "_misc"
}
```

| Field | Default | Description |
|---|---|---|
| `links_folder` | (none, required) | Root of the link tree |
| `link_kind` | OS default (see below) | Link type: `symlink`, `finder-alias` (macOS), `junction` (Windows) |
| `link_title_format` | `{{.Title}}` | Go template for link names. Fields: `.Title`, `.ID`. Supports functions: `date`, `upper`, `lower`, `join`. Old `{token}` syntax is auto-migrated |
| `link_sink_name` | (empty = disabled) | Name of sink folders in the tree |

## Link kinds

Different operating systems offer different ways to represent a link to a
directory. `prj` picks the kind that requires no privileges by default.

| Kind | OS | Privileges | Shell-traversable | Explorer/Finder | Cross-volume | Target must exist |
|---|---|---|---|---|---|---|
| `symlink` | all | Windows: Dev Mode or admin | yes | yes | yes | no |
| `finder-alias` | macOS | none | no | yes | yes | yes |
| `junction` | Windows | none | yes | yes | **no** | yes |

**OS defaults:**
- macOS, Linux, FreeBSD: `symlink`
- Windows: `junction` (chosen because it needs no privileges, unlike symlinks
  which require Developer Mode or an admin shell)

### Fallback rule

When the configured kind cannot be created for a given link, `prj link`
falls back to `symlink` rather than failing. There is one rule with three
cases:

- `finder-alias` and `junction` both need the target to exist (alias
  bookmarks are computed from the target; junctions are reparse points
  on a real directory). For metadata-only or not-yet-synced projects,
  the link becomes a (broken) symlink — it works as soon as the target
  syncs.
- `junction` additionally requires `links_folder` and the target to be
  on the same volume (NTFS junctions cannot span drives). Cross-volume
  links downgrade to `symlink` per-link, even when `link_kind: junction`
  is configured globally.
- `symlink` is always feasible at the algorithm level. On Windows it may
  still fail at creation time if Developer Mode is off and the user
  isn't running elevated — see below.

### Windows: handling the privilege error

Windows symlink creation requires the `SeCreateSymbolicLinkPrivilege`,
which is granted only to admins or — since Windows 10 1703 — to any user
when Developer Mode is enabled. If `prj` cannot create a symlink because
of this, it surfaces a contextual message:

- **Explicit `link_kind: symlink`**: recommends switching to junctions
  via `prj config set link_kind junction`.
- **Cross-volume fallback** (junction was wanted, link/target on
  different drives, symlink also blocked): recommends putting
  `links_folder` on the same volume as projects, or enabling Developer
  Mode.

The default Windows configuration (`junction`, both folders on the same
volume) needs no privilege at all.

## Which projects get links

Every known project (has folder or metadata) gets placed. Behavior depends
on tags:

- **Has tags, matches branches**: placed in each matching branch's deepest folder
- **Has tags, no branch matches**: placed in sink (if configured)
- **No tags**: placed in sink (if configured)
- **No tags, no sink**: no links (orphan)

Projects without a local folder are skipped by default.
Use `--all` to include metadata-only projects. When the configured link
kind needs the target to exist (`finder-alias`, `junction`), missing
targets fall back to symlinks. The symlinks work automatically once the
project folder syncs. See [Link kinds](#link-kinds) for the full rule.

Exclusion from the links tree is not controlled by absence of tags. If
needed in the future, a dedicated mechanism (e.g. `nolink` tag) would
handle that. Tags serve classification, not visibility control.

## Folder tag derivation

Folder names are parsed into tags:

```
"Programming"       → {programming}
"Photo & Video"     → {photo, video}
"ML & AI"           → {ml, ai}
"C++"               → {c++}
"ACME Inc"          → {acme_inc}
```

Rules:
- Split on ` & ` (space-ampersand-space) into segments
- Each segment: lowercase, collapse whitespace to `_`, strip outer `_`
- Special characters (`+`, `#`, `$`, etc.) preserved
- `AT&T` stays `{at&t}` (no spaces around `&`, no split)

Folder tags are local — a child does not inherit parent tags. Matching
is per-folder at each level independently.

## Placement algorithm

### Core: deepest-match traversal

For each project, starting at the links root:

1. Check each top-level child: does any of its tags overlap the project's
   tags? (OR — any single match is enough.)
2. For each matching child, recurse: check *its* children the same way.
3. When a folder matches but none of its children do — that folder is a
   **placement target** (the deepest relevant point).
4. Root is never a target (depth > 0 guard).

A project with tags `{cli, golang}` in a tree like:

```
Links/
  Programming/           → {programming}
    golang/              → {golang}
    python/              → {python}
  Work/
    cli/                 → {cli}
      automation/        → {automation}
```

Gets placed in `Programming/golang/` and `Work/cli/`. Not in
`Programming/` (deeper match exists), not in `Work/cli/automation/`
(tag `automation` not in project's tags).

### Multi-branch placement

The algorithm walks all top-level branches independently. A project can
land in multiple branches — this is by design. The links tree is a
multi-dimensional view, not a single taxonomy.

### Sink folders

When `link_sink_name` is configured (e.g. `"_misc"`):

**Branch sink**: algorithm reaches a folder where no children match. If a
direct child named `_misc` exists, place there instead of in the folder
itself. Keeps categorized folders clean — only sub-categorized projects
appear at that level, the rest go to `_misc`.

```
Programming/           → {programming}    project with tags {programming}
  golang/              → {golang}         lands here if tagged {golang}
  python/              → {python}         lands here if tagged {python}
  _misc/                                  ← lands here if tagged {programming}
                                            but not {golang} or {python}
```

**Root sink**: no top-level branch matches at all → if `_misc` exists at
links root, place there. Catches unmatched and tagless projects.

**Sink tag stripping**: if a project has a tag matching the sink name,
that tag is ignored during matching. Prevents gaming (tagging a project
`_misc` should not affect placement logic).

**Disabled**: when `link_sink_name` is empty, sinks are not used.
Unmatched/tagless projects get no links. Folders where no children
match become targets themselves (no redirection).

### Algorithm invariants

1. **Deepest match**: always return the deepest folder where matching
   terminates, never intermediate ancestors
2. **Per-folder matching**: tags are local; no inheritance from parent
3. **OR semantics**: any single tag overlap passes the gate
4. **Root exclusion**: root is never a placement target
5. **Sink atomicity**: either found targets OR sink, never both
6. **Deterministic**: same tags + same tree = same placements, always

## Link naming

Link names are derived from project metadata:

| Situation | Name |
|---|---|
| Has title, unique in folder | Per `link_title_format` (default: `{{.Title}}`) |
| Has title, collides in folder | `Title (id)` for ALL colliders |
| No title | `id` |

When collision exists, all colliding projects get the `(id)` suffix —
not just the "second" one. This prevents non-deterministic naming based
on processing order.

If the format already contains `.ID` (e.g. `{{.ID}} {{.Title}}`),
collision suffixing is skipped — the ID in the name already guarantees
uniqueness.

### Filename sanitization

Link names are sanitized for cross-platform safety. Because files sync
between machines via Resilio Sync, the strictest union of all platform
rules is always applied:

- Forbidden characters replaced with `-`: `/ \ : * ? " < > |` and
  control chars (0x00-0x1F)
- Trailing dots and spaces trimmed (Windows silently strips them)
- Windows reserved device names prefixed with `_`: `CON`, `PRN`, `AUX`,
  `NUL`, `COM1-9`, `LPT1-9` (case-insensitive, with or without extension)
- Truncated to 255 bytes at a UTF-8 rune boundary
- If sanitization reduces the name to only dashes, falls back to project ID

## Reconciliation

`prj link` computes the desired state (placements + names), then
reconciles against the actual state of the links folder.

### Detecting managed links

A link in the tree is "ours" if its target resolves into
`projects_folder/<valid-id>`. This is checked by:
1. Resolving the link (symlink readlink, alias resolution)
2. Checking the target is inside `projects_folder`
3. Validating the final path component is a valid project ID

No manifest file. The target *is* the identity.

### Reconciliation matrix

| Existing | Desired | Action |
|---|---|---|
| nothing | link here | **create** |
| our link, right target, right kind | link here | **skip** |
| our link, right target, wrong kind | link here | **replace** (remove + create) |
| our link, wrong location | link elsewhere | **remove** here, **create** there |
| our link | no placement | **remove** |
| not our link (foreign target) | link here | **conflict** (warn, skip) |
| regular file/folder | link here | **conflict** (warn, skip) |

"Wrong kind" handles switching between any two kinds: change `link_kind`
in config, re-run `prj link`, all links are recreated with the new type.
Examples: `finder-alias` ↔ `symlink` on macOS, `symlink` → `junction` on
Windows after enabling the new default.

The kind compared against is the *effective* kind for that specific
link, not the raw config value. A project on a different volume from
`links_folder` has its effective kind degraded from `junction` to
`symlink` (see [Link kinds](#link-kinds)) — an existing symlink there is
correct and won't churn.

### Unicode normalization (macOS)

macOS filesystems return filenames in NFD (decomposed Unicode), while Go
strings use NFC (composed). A link named `Настройка` has different byte
representations in the desired state (NFC, from metadata) vs the actual
state (NFD, from filesystem scan). Direct string comparison fails.

Reconciliation handles this by matching on `(folder, projectID)` pairs
as a fallback when exact path matching misses. The project ID is ASCII,
so it's unaffected by normalization. This avoids the need for a Unicode
normalization library.

### Parent directory creation

When creating a link, the parent directory is created if it doesn't exist
(`MkdirAll`). This handles the case where a sink folder was removed (all
its links were deleted in a previous run) and needs to be recreated.

### Cleanup of empty folders

After removing managed links, if a folder becomes empty, it is NOT
removed. The tree structure is user-managed. Empty folders might be
intentional placeholders.

## Testing strategy

### Layer 1: Placement algorithm (pure logic)

Go table-driven tests with a tree builder DSL. No filesystem. Tests
build in-memory trees and assert placements. ~21 scenarios covering:
single/multi branch, deepest match, sink at branch/root, sink tag
stripping, tagless projects, OR semantics, inaccessible branches.

### Layer 2: Naming + sanitization (pure logic)

Table-driven tests for link name formatting, collision resolution,
cross-platform filename sanitization (forbidden chars, Windows reserved
names, UTF-8 truncation), and sanitize-to-empty fallback.

### Layer 3: Scan + reconciliation (filesystem)

Go tests with `t.TempDir()` and real symlinks. Covers: managed link
detection, foreign link/file ignoring, broken symlinks, nested depths,
filter by project ID, NFD/NFC Unicode normalization, reconciliation
matrix (create/remove/replace/conflict/skip), parent dir creation.

### Layer 4: Tree building (filesystem)

Tests `BuildTree` with real directory structures. Verifies hidden dir
skipping, file ignoring, tag derivation, path assignment.

## Deferred work

- `.tags` file in folders to override/extend derived tags
- `nolink` tag or config to exclude specific projects from link tree
- Link icon management (faded icons for missing targets)
- Watcher mode (auto-sync on tag or tree changes)
