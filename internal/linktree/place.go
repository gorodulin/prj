package linktree

// FindPlacements returns target folders where a project's link should be placed.
// sinkName empty disables sink behavior.
func FindPlacements(root *Folder, projectTags []string, sinkName string) []*Folder {
	tags := toSet(projectTags)
	if sinkName != "" {
		delete(tags, sinkName)
	}

	var targets []*Folder
	for _, child := range root.Children {
		if matchesAny(child.Tags, tags) {
			targets = append(targets, descend(child, tags, sinkName)...)
		}
	}

	if len(targets) > 0 {
		return targets
	}

	// Root sink fallback: catches unmatched and tagless projects.
	if sinkName != "" {
		if sink := findChild(root, sinkName); sink != nil {
			return []*Folder{sink}
		}
	}

	return nil
}

// descend recursively finds the deepest matching folders in a branch.
func descend(folder *Folder, tags map[string]bool, sinkName string) []*Folder {
	var reachable []*Folder
	for _, child := range folder.Children {
		if matchesAny(child.Tags, tags) {
			reachable = append(reachable, child)
		}
	}

	if len(reachable) > 0 {
		var targets []*Folder
		for _, child := range reachable {
			targets = append(targets, descend(child, tags, sinkName)...)
		}
		return targets
	}

	// No children match. Redirect to sink if present, otherwise self.
	if sinkName != "" {
		if sink := findChild(folder, sinkName); sink != nil {
			return []*Folder{sink}
		}
	}

	return []*Folder{folder}
}

// matchesAny reports whether any folder tag is in the project tag set.
func matchesAny(folderTags []string, projectTags map[string]bool) bool {
	for _, t := range folderTags {
		if projectTags[t] {
			return true
		}
	}
	return false
}

// findChild returns the direct child with the given name, or nil.
func findChild(folder *Folder, name string) *Folder {
	for _, c := range folder.Children {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func toSet(tags []string) map[string]bool {
	s := make(map[string]bool, len(tags))
	for _, t := range tags {
		s[t] = true
	}
	return s
}
