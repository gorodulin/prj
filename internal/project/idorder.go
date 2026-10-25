package project

import "strings"

// CompareIDs orders project IDs by creation, returning -1, 0 or +1 like
// cmp.Compare. ULID, UUIDv7 and KSUID are fixed-length and lexically
// time-ordered. aYYYYMMDDb IDs compare by prefix and date first, then by
// suffix, where the base-26 sequence (z < aa) makes shorter mean earlier.
func CompareIDs(a, b string) int {
	pat := idPatterns[FormatAYMDb]
	if !pat.MatchString(a) || !pat.MatchString(b) {
		return strings.Compare(a, b)
	}
	headA, suffixA := splitAYMDbSuffix(a)
	headB, suffixB := splitAYMDbSuffix(b)
	if c := strings.Compare(headA, headB); c != 0 {
		return c
	}
	if len(suffixA) != len(suffixB) {
		if len(suffixA) < len(suffixB) {
			return -1
		}
		return 1
	}
	return strings.Compare(suffixA, suffixB)
}

// splitAYMDbSuffix splits an aYYYYMMDDb ID after its last date digit.
func splitAYMDbSuffix(id string) (head, suffix string) {
	i := strings.LastIndexAny(id, "0123456789") + 1
	return id[:i], id[i:]
}
