package metadata

// Meta holds resolved title and tags for a project.
type Meta struct {
	Title string
	Tags  []string
}

// FindHeads returns snapshots that are not referenced as a parent by any other.
// These are the "leaf" nodes of the DAG — the current state(s).
func FindHeads(snapshots []Snapshot) []Snapshot {
	referenced := make(map[string]bool)
	for _, s := range snapshots {
		for _, parent := range s.BasedOn {
			referenced[parent] = true
		}
	}

	var heads []Snapshot
	for _, s := range snapshots {
		if !referenced[s.Filename] {
			heads = append(heads, s)
		}
	}
	return heads
}

// TagDeltas computes added and removed tags between old and new tag sets.
// Both inputs must be normalized (sorted, deduplicated, lowercase).
func TagDeltas(oldTags, newTags []string) (added, removed []string) {
	oldSet := make(map[string]bool, len(oldTags))
	for _, t := range oldTags {
		oldSet[t] = true
	}
	newSet := make(map[string]bool, len(newTags))
	for _, t := range newTags {
		newSet[t] = true
	}
	for _, t := range newTags {
		if !oldSet[t] {
			added = append(added, t)
		}
	}
	for _, t := range oldTags {
		if !newSet[t] {
			removed = append(removed, t)
		}
	}
	return added, removed
}

// LatestHead returns the Meta from the most recent head snapshot.
// With sorted snapshots, the latest head has the greatest filename.
// Returns zero Meta if snapshots is empty.
func LatestHead(snapshots []Snapshot) Meta {
	if len(snapshots) == 0 {
		return Meta{}
	}

	heads := FindHeads(snapshots)
	if len(heads) == 0 {
		// All snapshots reference each other (shouldn't happen).
		// Fall back to the last snapshot by timestamp.
		heads = snapshots[len(snapshots)-1:]
	}

	latest := heads[len(heads)-1]

	// Resolve title: use the latest head's title_set if explicit,
	// otherwise scan all snapshots in reverse for the most recent explicit title.
	var title string
	if latest.TitleSet != nil {
		title = *latest.TitleSet
	} else {
		for i := len(snapshots) - 1; i >= 0; i-- {
			if snapshots[i].TitleSet != nil {
				title = *snapshots[i].TitleSet
				break
			}
		}
	}

	return Meta{
		Title: title,
		Tags:  latest.Tags,
	}
}
