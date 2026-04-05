package project

import (
	"strings"
	"time"
)

// ParseIDTime extracts the creation timestamp from a project ID.
// It auto-detects the ID format using the known regex patterns.
// For aYYYYMMDDb, the returned time is midnight UTC on the embedded date.
// For ULID/UUIDv7/KSUID, the returned time is the exact encoded timestamp (UTC).
// Returns zero time and false if the ID format is not recognized or parsing fails.
func ParseIDTime(id string) (time.Time, bool) {
	for format, pat := range idPatterns {
		if !pat.MatchString(id) {
			continue
		}
		switch format {
		case FormatAYMDb:
			return parseAYMDbTime(id)
		case FormatULID:
			return parseULIDTime(id)
		case FormatUUIDv7:
			return parseUUIDv7Time(id)
		case FormatKSUID:
			return parseKSUIDTime(id)
		}
	}
	return time.Time{}, false
}

func parseAYMDbTime(id string) (time.Time, bool) {
	// Extract 8-digit date from after the letter prefix.
	// Pattern: 1-2 letters + 8 digits + 1-3 letters.
	start := 0
	for start < len(id) && id[start] >= 'a' && id[start] <= 'z' {
		start++
	}
	// Skip optional separator (- or _) between prefix and date.
	if start < len(id) && (id[start] == '-' || id[start] == '_') {
		start++
	}
	if start == 0 || start+8 > len(id) {
		return time.Time{}, false
	}
	dateStr := id[start : start+8]
	t, err := time.Parse("20060102", dateStr)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func parseULIDTime(id string) (time.Time, bool) {
	if len(id) != 26 {
		return time.Time{}, false
	}
	// First 10 chars are Crockford Base32 encoding of 48-bit ms timestamp.
	var ms uint64
	for i := 0; i < 10; i++ {
		v := crockfordValue(id[i])
		if v < 0 {
			return time.Time{}, false
		}
		ms = (ms << 5) | uint64(v)
	}
	return time.UnixMilli(int64(ms)).UTC(), true
}

func parseUUIDv7Time(id string) (time.Time, bool) {
	// Remove dashes to get 32 hex chars.
	hex := strings.ReplaceAll(id, "-", "")
	if len(hex) != 32 {
		return time.Time{}, false
	}
	// First 12 hex chars = 48-bit ms timestamp.
	var ms uint64
	for i := 0; i < 12; i++ {
		v := hexValue(hex[i])
		if v < 0 {
			return time.Time{}, false
		}
		ms = (ms << 4) | uint64(v)
	}
	return time.UnixMilli(int64(ms)).UTC(), true
}

const ksuidEpoch = 1400000000 // 2014-05-13T16:53:20Z

func parseKSUIDTime(id string) (time.Time, bool) {
	if len(id) != 27 {
		return time.Time{}, false
	}
	// Decode base62 to 20 bytes, first 4 are timestamp.
	const base62 = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	var decoded [20]byte
	src := make([]int, 27)
	for i, c := range id {
		idx := strings.IndexByte(base62, byte(c))
		if idx < 0 {
			return time.Time{}, false
		}
		src[i] = idx
	}
	// Base62 decode: convert 27-digit base62 number to 20-byte big-endian.
	for i := 0; i < 20; i++ {
		var remainder int
		for j := 0; j < len(src); j++ {
			acc := remainder*62 + src[j]
			src[j] = acc / 256
			remainder = acc % 256
		}
		decoded[i] = byte(remainder)
	}
	// Reverse: the loop fills least-significant byte first.
	for i, j := 0, 19; i < j; i, j = i+1, j-1 {
		decoded[i], decoded[j] = decoded[j], decoded[i]
	}
	ts := uint32(decoded[0])<<24 | uint32(decoded[1])<<16 | uint32(decoded[2])<<8 | uint32(decoded[3])
	return time.Unix(int64(ts)+ksuidEpoch, 0).UTC(), true
}

func crockfordValue(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'A' && c <= 'H':
		return int(c-'A') + 10
	case c == 'J', c == 'K':
		return int(c-'J') + 18
	case c >= 'M' && c <= 'N':
		return int(c-'M') + 20
	case c >= 'P' && c <= 'T':
		return int(c-'P') + 22
	case c >= 'V' && c <= 'Z':
		return int(c-'V') + 27
	default:
		return -1
	}
}

func hexValue(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	default:
		return -1
	}
}
