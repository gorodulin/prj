package project

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"time"
)

// GenerateID creates a new unique project ID based on the given format.
// It checks against existing IDs to avoid collisions.
func GenerateID(idFormat string, existing []string) (string, error) {
	switch idFormat {
	case FormatAYMDb:
		return generateAYMDb(existing), nil
	case FormatUUIDv7:
		return generateUUIDv7()
	case FormatULID:
		return generateULID()
	case FormatKSUID:
		return generateKSUID()
	default:
		return "", fmt.Errorf("ID generation not implemented for format %q", idFormat)
	}
}

func generateAYMDb(existing []string) string {
	existingSet := make(map[string]bool, len(existing))
	for _, id := range existing {
		existingSet[id] = true
	}

	dateStr := time.Now().UTC().Format("20060102")
	prefix := "p"

	for i := 0; ; i++ {
		candidate := prefix + dateStr + base26Suffix(i)
		if !existingSet[candidate] {
			return candidate
		}
	}
}

// generateUUIDv7 creates a UUIDv7: 48-bit unix ms timestamp + 4-bit version (7)
// + 12-bit random + 2-bit variant (10) + 62-bit random.
func generateUUIDv7() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate UUIDv7: %w", err)
	}

	ms := uint64(time.Now().UnixMilli())
	b[0] = byte(ms >> 40)
	b[1] = byte(ms >> 32)
	b[2] = byte(ms >> 24)
	b[3] = byte(ms >> 16)
	b[4] = byte(ms >> 8)
	b[5] = byte(ms)

	b[6] = (b[6] & 0x0F) | 0x70 // version 7
	b[8] = (b[8] & 0x3F) | 0x80 // variant 10

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		binary.BigEndian.Uint32(b[0:4]),
		binary.BigEndian.Uint16(b[4:6]),
		binary.BigEndian.Uint16(b[6:8]),
		binary.BigEndian.Uint16(b[8:10]),
		b[10:16],
	), nil
}

// generateULID creates a ULID: 48-bit unix ms timestamp (Crockford base32)
// + 80-bit random (Crockford base32). 26 chars total.
func generateULID() (string, error) {
	var b [10]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate ULID: %w", err)
	}

	const crockford = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
	ms := uint64(time.Now().UnixMilli())

	var out [26]byte
	// Timestamp: 10 chars of base32 for 48-bit value
	out[0] = crockford[(ms>>45)&0x1F]
	out[1] = crockford[(ms>>40)&0x1F]
	out[2] = crockford[(ms>>35)&0x1F]
	out[3] = crockford[(ms>>30)&0x1F]
	out[4] = crockford[(ms>>25)&0x1F]
	out[5] = crockford[(ms>>20)&0x1F]
	out[6] = crockford[(ms>>15)&0x1F]
	out[7] = crockford[(ms>>10)&0x1F]
	out[8] = crockford[(ms>>5)&0x1F]
	out[9] = crockford[ms&0x1F]

	// Randomness: 16 chars of base32 for 80 bits
	// Pack 10 random bytes into 80 bits, extract 5-bit groups
	bits := binary.BigEndian.Uint64(b[0:8])
	tail := uint64(b[8])<<8 | uint64(b[9])
	out[10] = crockford[(bits>>59)&0x1F]
	out[11] = crockford[(bits>>54)&0x1F]
	out[12] = crockford[(bits>>49)&0x1F]
	out[13] = crockford[(bits>>44)&0x1F]
	out[14] = crockford[(bits>>39)&0x1F]
	out[15] = crockford[(bits>>34)&0x1F]
	out[16] = crockford[(bits>>29)&0x1F]
	out[17] = crockford[(bits>>24)&0x1F]
	out[18] = crockford[(bits>>19)&0x1F]
	out[19] = crockford[(bits>>14)&0x1F]
	out[20] = crockford[(bits>>9)&0x1F]
	out[21] = crockford[(bits>>4)&0x1F]
	// Last 4 bits of first uint64 + first bit of tail
	out[22] = crockford[((bits&0x0F)<<1)|((tail>>15)&0x01)]
	out[23] = crockford[(tail>>10)&0x1F]
	out[24] = crockford[(tail>>5)&0x1F]
	out[25] = crockford[tail&0x1F]

	return string(out[:]), nil
}

// generateKSUID creates a KSUID: 4-byte timestamp (seconds since epoch 2014-05-13)
// + 16-byte random, encoded as 27-char base62.
func generateKSUID() (string, error) {
	const ksuidEpoch = 1400000000 // 2014-05-13T16:53:20Z

	var payload [20]byte
	ts := uint32(time.Now().Unix() - ksuidEpoch)
	binary.BigEndian.PutUint32(payload[0:4], ts)
	if _, err := rand.Read(payload[4:]); err != nil {
		return "", fmt.Errorf("generate KSUID: %w", err)
	}

	// Base62 encode 20 bytes → 27 chars
	const base62 = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	var result [27]byte
	// Convert payload to big integer via repeated division
	src := make([]byte, 20)
	copy(src, payload[:])
	for i := 26; i >= 0; i-- {
		var remainder int
		for j := 0; j < len(src); j++ {
			acc := remainder*256 + int(src[j])
			src[j] = byte(acc / 62)
			remainder = acc % 62
		}
		result[i] = base62[remainder]
	}

	return string(result[:]), nil
}

// base26Suffix converts a zero-based index to a base-26 letter sequence:
// 0→a, 1→b, ..., 25→z, 26→aa, 27→ab, ...
func base26Suffix(n int) string {
	n++ // 1-based for proper a..z, aa..az sequence
	var result []byte
	for n > 0 {
		n--
		result = append(result, byte('a'+n%26))
		n /= 26
	}
	// Reverse
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return string(result)
}
