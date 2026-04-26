package linktree

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/gorodulin/prj/internal/config"
	"github.com/gorodulin/prj/internal/platform"
	"github.com/gorodulin/prj/internal/project"
)

// ActionKind describes what reconciliation decided for a link path.
type ActionKind int

const (
	ActionCreate  ActionKind = iota // link missing, create it
	ActionRemove                    // stale link, remove it
	ActionReplace                   // right target, wrong kind — remove + create
	ActionConflict                  // non-link or foreign link blocks desired path
	ActionSkip                      // already correct
)

// Action is a single reconciliation step.
type Action struct {
	Kind    ActionKind
	Path    string // full link path (for remove/replace: the existing path)
	NewPath string // for replace with rename: the desired new path
	Target  string // project folder path (empty for remove)
	ID      string // project ID
	Detail  string // reason for conflict or replace
}

// DesiredLink pairs a project folder path with its ID.
type DesiredLink struct {
	Target string
	ID     string
}

// Reconcile computes actions to sync desired state against actual.
// desired maps full link path → DesiredLink.
// linkKind is one of "symlink", "finder-alias", or "junction".
// projectsFolder is used to detect foreign project links for automatic
// conflict resolution. Pass "" to disable this behavior.
func Reconcile(desired map[string]DesiredLink, actual []ManagedLink, linkKind, projectsFolder string) []Action {
	actualByPath := make(map[string]ManagedLink, len(actual))
	// Index by (folder, projectID) to handle Unicode normalization differences
	// between Go strings (NFC) and macOS filenames (NFD).
	type folderProject struct{ folder, id string }
	actualByFolderID := make(map[folderProject]ManagedLink, len(actual))
	for _, ml := range actual {
		actualByPath[ml.Path] = ml
		actualByFolderID[folderProject{filepath.Dir(ml.Path), ml.ProjectID}] = ml
	}

	var actions []Action

	// Check each desired link against actual state.
	for desiredPath, dl := range desired {
		ml, exists := actualByPath[desiredPath]
		renamed := false
		if !exists {
			// Path didn't match exactly — try matching by folder + project ID.
			// This catches renames (format change) and macOS NFD/NFC
			// Unicode normalization mismatches in filenames.
			key := folderProject{filepath.Dir(desiredPath), dl.ID}
			if mlAlt, found := actualByFolderID[key]; found {
				ml = mlAlt
				exists = true
				// Detect genuine renames (format change) but not Unicode
				// normalization mismatches (NFC vs NFD). macOS stores
				// filenames in NFD; Go strings are NFC. Compare only the
				// ASCII bytes of the base names: NFC/NFD only affects
				// non-ASCII characters, so if the ASCII skeletons differ,
				// it's a real rename.
				renamed = asciiKey(filepath.Base(mlAlt.Path)) != asciiKey(filepath.Base(desiredPath))
			}
		}
		if !exists {
			// Check if blocker is a foreign project link we can work around.
			if altPath, ok := retryWithIDSuffix(desiredPath, dl, projectsFolder); ok {
				if checkBlocker(altPath) == "" {
					actions = append(actions, Action{
						Kind:   ActionCreate,
						Path:   altPath,
						Target: dl.Target,
						ID:     dl.ID,
					})
					continue
				}
			}
			// Nothing at this path, or non-project blocker.
			if blocker := checkBlocker(desiredPath); blocker != "" {
				actions = append(actions, Action{
					Kind:   ActionConflict,
					Path:   desiredPath,
					Target: dl.Target,
					ID:     dl.ID,
					Detail: blocker,
				})
				continue
			}
			actions = append(actions, Action{
				Kind:   ActionCreate,
				Path:   desiredPath,
				Target: dl.Target,
				ID:     dl.ID,
			})
			continue
		}

		// Link exists (exact path or folder+ID match). Check if it needs updating.
		// effectiveLinkKind handles fallbacks: finder-alias and junction need the
		// target to exist; junction additionally requires same-volume.
		effectiveKind := effectiveLinkKind(linkKind, dl.Target, desiredPath)
		if ml.ProjectID == dl.ID && ml.Kind == effectiveKind && !renamed {
			actions = append(actions, Action{
				Kind: ActionSkip,
				Path: desiredPath,
				ID:   dl.ID,
			})
		} else if ml.ProjectID == dl.ID && (ml.Kind != effectiveKind || renamed) {
			// Right target, but wrong kind or wrong name — replace.
			detail := ""
			if renamed {
				detail = "renamed"
			} else {
				detail = fmt.Sprintf("wrong kind: want %s", effectiveKind)
			}
			actions = append(actions, Action{
				Kind:    ActionReplace,
				Path:    ml.Path, // use actual filesystem path for removal
				NewPath: desiredPath,
				Target:  dl.Target,
				ID:      dl.ID,
				Detail:  detail,
			})
		} else {
			// Wrong target — remove old, create new.
			actions = append(actions, Action{
				Kind: ActionRemove,
				Path: ml.Path,
				ID:   ml.ProjectID,
			})
			actions = append(actions, Action{
				Kind:   ActionCreate,
				Path:   desiredPath,
				Target: dl.Target,
				ID:     dl.ID,
			})
		}

		delete(actualByPath, ml.Path)
		delete(actualByFolderID, folderProject{filepath.Dir(ml.Path), ml.ProjectID})
	}

	// Remaining actual links not in desired → stale, remove.
	for _, ml := range actualByPath {
		actions = append(actions, Action{
			Kind: ActionRemove,
			Path: ml.Path,
			ID:   ml.ProjectID,
		})
	}

	return actions
}

// retryWithIDSuffix checks whether desiredPath is blocked by a foreign
// project link (a symlink/alias pointing into projectsFolder whose target
// basename is a recognized project ID). If so, it returns an alternative
// path with " (<ID>)" appended. Returns ("", false) if the blocker is not
// a foreign project link or projectsFolder is empty.
func retryWithIDSuffix(desiredPath string, dl DesiredLink, projectsFolder string) (string, bool) {
	if projectsFolder == "" {
		return "", false
	}
	target, _, ok := resolveTarget(desiredPath)
	if !ok {
		return "", false
	}
	// Reuse extractProjectID with empty format to check "direct child of projectsFolder".
	rel, ok := extractProjectID(target, projectsFolder, "", "")
	if !ok || !project.IsAnyValidID(rel) {
		return "", false
	}
	dir := filepath.Dir(desiredPath)
	base := filepath.Base(desiredPath)
	return filepath.Join(dir, base+" ("+dl.ID+")"), true
}

// checkBlocker returns a description if something non-link blocks a path,
// or empty string if the path is clear.
func checkBlocker(path string) string {
	info, err := os.Lstat(path)
	if err != nil {
		return "" // doesn't exist, path is clear
	}
	if info.IsDir() {
		return "blocked by directory"
	}
	if info.Mode().IsRegular() {
		return "blocked by regular file"
	}
	// Could be a symlink or alias not detected as managed — treat as foreign.
	return "blocked by existing file"
}

// asciiKey extracts only ASCII bytes from s. Used to compare filenames
// without being affected by Unicode NFC/NFD normalization differences.
func asciiKey(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] < 0x80 {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// Apply executes actions on the filesystem.
// Skips ActionConflict and ActionSkip.
func Apply(actions []Action, linkKind string) error {
	for _, a := range actions {
		switch a.Kind {
		case ActionCreate:
			if err := createLink(a.Path, a.Target, linkKind); err != nil {
				return fmt.Errorf("create link %s: %w", a.Path, err)
			}
		case ActionRemove:
			if err := os.Remove(a.Path); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("remove link %s: %w", a.Path, err)
			}
		case ActionReplace:
			if err := os.Remove(a.Path); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("remove link %s for replace: %w", a.Path, err)
			}
			createPath := a.Path
			if a.NewPath != "" {
				createPath = a.NewPath
			}
			if err := createLink(createPath, a.Target, linkKind); err != nil {
				return fmt.Errorf("create link %s after replace: %w", createPath, err)
			}
		case ActionConflict, ActionSkip:
			// No filesystem action.
		}
	}
	return nil
}

// effectiveLinkKind reports the kind that will actually be created at linkPath
// given the configured want and the target's state. The rule is single: if the
// wanted kind is impossible, fall back to symlink.
//   - finder-alias and junction both need the target to exist.
//   - junction additionally requires linkPath and target to be on the same volume.
func effectiveLinkKind(want, target, linkPath string) string {
	if want == config.LinkKindSymlink {
		return want
	}
	if _, err := os.Stat(target); err != nil {
		return config.LinkKindSymlink
	}
	if want == config.LinkKindJunction &&
		filepath.VolumeName(linkPath) != filepath.VolumeName(target) {
		return config.LinkKindSymlink
	}
	return want
}

func createLink(path, target, linkKind string) error {
	// Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create parent dir: %w", err)
	}

	switch effectiveLinkKind(linkKind, target, path) {
	case config.LinkKindFinderAlias:
		return platform.CreateAlias(path, target)
	case config.LinkKindJunction:
		return platform.CreateJunction(path, target)
	default:
		return wrapSymlinkError(os.Symlink(target, path), path, target, linkKind)
	}
}

// wrapSymlinkError converts a Windows ERROR_PRIVILEGE_NOT_HELD into a typed
// SymlinkPrivilegeError so the cmd layer can format a tailored message. The
// FellBackFromJunction flag captures whether the caller actually wanted a
// junction but had to fall back (cross-volume case).
func wrapSymlinkError(err error, path, target, configuredKind string) error {
	if err == nil {
		return nil
	}
	if !errors.Is(err, syscall.Errno(privilegeNotHeld)) {
		return err
	}
	return &platform.SymlinkPrivilegeError{
		LinkPath:             path,
		Target:               target,
		FellBackFromJunction: configuredKind == config.LinkKindJunction,
	}
}

// privilegeNotHeld is Windows ERROR_PRIVILEGE_NOT_HELD. Defined as a constant
// rather than imported from x/sys/windows to avoid a dependency for one value.
const privilegeNotHeld = 1314
