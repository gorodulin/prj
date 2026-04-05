package project

import (
	"regexp"
	"strings"
)

// Supported project ID format names.
const (
	FormatAYMDb = "aYYYYMMDDb"
	FormatUUIDv7 = "UUIDv7"
	FormatULID  = "ULID"
	FormatKSUID = "KSUID"
)

var idPatterns = map[string]*regexp.Regexp{
	FormatAYMDb:  regexp.MustCompile(`^[a-z]{1,5}[-_]?\d{8}[a-z]{1,3}$`),
	FormatUUIDv7: regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-7[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`),
	FormatULID:   regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}$`),
	FormatKSUID:  regexp.MustCompile(`^[0-9A-Za-z]{27}$`),
}

// validPrefix matches a valid aYYYYMMDDb prefix: 1-5 lowercase letters
// with an optional trailing separator (- or _).
var validPrefix = regexp.MustCompile(`^[a-z]{1,5}[-_]?$`)

// aYMDbSuffix matches the date+suffix portion of an aYYYYMMDDb ID (after the prefix).
var aYMDbSuffix = regexp.MustCompile(`^\d{8}[a-z]{1,3}$`)

// IsValidPrefix reports whether s is a valid aYYYYMMDDb prefix.
func IsValidPrefix(s string) bool {
	return validPrefix.MatchString(s)
}

// IsAnyValidID reports whether name matches any known project ID format
// (any prefix for aYYYYMMDDb, any format). Used to detect foreign project
// links without knowing the configured format or prefix.
func IsAnyValidID(name string) bool {
	for _, pat := range idPatterns {
		if pat.MatchString(name) {
			return true
		}
	}
	return false
}

// IsValidID reports whether name matches the given project ID format.
// For aYYYYMMDDb, prefix is the required leading sequence (e.g. "prj", "p-").
// For other formats, prefix is ignored.
func IsValidID(name, format, prefix string) bool {
	if format == FormatAYMDb {
		if !strings.HasPrefix(name, prefix) {
			return false
		}
		return aYMDbSuffix.MatchString(name[len(prefix):])
	}
	pat, ok := idPatterns[format]
	if !ok {
		return false
	}
	return pat.MatchString(name)
}
