package project

import (
	"fmt"
	"os"
	"strings"
)

// CollectIDsFromFolder scans a directory for subdirectories matching the given
// ID format and returns their project IDs. Hidden directories are skipped.
//
// If suffix is non-empty, directory names must end with that suffix, and it is
// stripped to produce the project ID (e.g. "p20250101a_meta" → "p20250101a").
//
// If idFormat is empty, all non-hidden directories are included (after suffix
// stripping if applicable).
func CollectIDsFromFolder(folder, idFormat, prefix, suffix string) ([]string, error) {
	entries, err := os.ReadDir(folder)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read folder %s: %w", folder, err)
	}

	var ids []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name[0] == '.' {
			continue
		}

		id := name
		if suffix != "" {
			if !strings.HasSuffix(name, suffix) {
				continue
			}
			id = name[:len(name)-len(suffix)]
		}

		if idFormat != "" && !IsValidID(id, idFormat, prefix) {
			continue
		}

		ids = append(ids, id)
	}

	return ids, nil
}
