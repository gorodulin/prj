package project

import "regexp"

// Supported project ID format names.
const (
	FormatAYMDb = "aYYYYMMDDb"
	FormatUUIDv7 = "UUIDv7"
	FormatULID  = "ULID"
	FormatKSUID = "KSUID"
)

var idPatterns = map[string]*regexp.Regexp{
	FormatAYMDb:  regexp.MustCompile(`^[a-z]{1,2}\d{8}[a-z]{1,3}$`),
	FormatUUIDv7: regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-7[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`),
	FormatULID:   regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}$`),
	FormatKSUID:  regexp.MustCompile(`^[0-9A-Za-z]{27}$`),
}

// IsValidID reports whether name matches the given project ID format.
func IsValidID(name, format string) bool {
	pat, ok := idPatterns[format]
	if !ok {
		return false
	}
	return pat.MatchString(name)
}
