package platform

import (
	"encoding/binary"
	"errors"
	"fmt"
	"unicode/utf16"
)

// Binary plist encoding/decoding for single string values.
// Used to store Finder comments in the com.apple.metadata:kMDItemFinderComment xattr.

const bplistHeader = "bplist00"

// EncodeBplistString encodes s as a binary plist containing a single string.
func EncodeBplistString(s string) []byte {
	var object []byte
	if isASCII(s) {
		object = encodeASCIIString([]byte(s))
	} else {
		object = encodeUnicodeString(s)
	}

	// Layout: header + object + offset_table + trailer
	objectOffset := len(bplistHeader)
	offsetTableOffset := objectOffset + len(object)

	// Offset table: single entry pointing to the object.
	offsetSize := intByteSize(uint64(objectOffset))
	offsetEntry := encodeUint(uint64(objectOffset), offsetSize)

	// 32-byte trailer.
	var trailer [32]byte
	// [0..5]  unused
	trailer[6] = byte(offsetSize)      // offsetIntSize
	trailer[7] = 1                     // objectRefSize (unused for single object)
	binary.BigEndian.PutUint64(trailer[8:16], 1)  // numObjects
	binary.BigEndian.PutUint64(trailer[16:24], 0)  // topObject
	binary.BigEndian.PutUint64(trailer[24:32], uint64(offsetTableOffset))

	buf := make([]byte, 0, len(bplistHeader)+len(object)+len(offsetEntry)+32)
	buf = append(buf, bplistHeader...)
	buf = append(buf, object...)
	buf = append(buf, offsetEntry...)
	buf = append(buf, trailer[:]...)
	return buf
}

// DecodeBplistString decodes a binary plist containing a single string.
func DecodeBplistString(data []byte) (string, error) {
	if len(data) < len(bplistHeader)+32 {
		return "", errors.New("bplist too short")
	}
	if string(data[:8]) != bplistHeader {
		return "", errors.New("not a binary plist")
	}

	// Read trailer (last 32 bytes).
	trailer := data[len(data)-32:]
	offsetIntSize := int(trailer[6])
	numObjects := binary.BigEndian.Uint64(trailer[8:16])
	offsetTableOffset := binary.BigEndian.Uint64(trailer[24:32])

	if numObjects != 1 {
		return "", fmt.Errorf("expected 1 object, got %d", numObjects)
	}
	if offsetIntSize == 0 || int(offsetTableOffset)+offsetIntSize > len(data)-32 {
		return "", errors.New("invalid offset table")
	}

	// Read the single offset.
	objOffset := decodeUint(data[offsetTableOffset:offsetTableOffset+uint64(offsetIntSize)])
	if int(objOffset) >= len(data)-32 {
		return "", errors.New("object offset out of range")
	}

	return decodeStringObject(data[objOffset:])
}

func decodeStringObject(data []byte) (string, error) {
	if len(data) == 0 {
		return "", errors.New("empty object")
	}

	typeTag := data[0] >> 4
	length := int(data[0] & 0x0f)
	pos := 1

	// Extended length.
	if length == 0x0f {
		if pos >= len(data) {
			return "", errors.New("truncated length")
		}
		intType := data[pos] >> 4
		if intType != 0x1 {
			return "", fmt.Errorf("expected int type for length, got 0x%x", intType)
		}
		intBytes := 1 << (data[pos] & 0x0f)
		pos++
		if pos+intBytes > len(data) {
			return "", errors.New("truncated length int")
		}
		length = int(decodeUint(data[pos : pos+intBytes]))
		pos += intBytes
	}

	switch typeTag {
	case 0x5: // ASCII string
		end := pos + length
		if end > len(data) {
			return "", errors.New("truncated ASCII string")
		}
		return string(data[pos:end]), nil

	case 0x6: // Unicode string (UTF-16BE), length is in code units
		byteLen := length * 2
		end := pos + byteLen
		if end > len(data) {
			return "", errors.New("truncated Unicode string")
		}
		units := make([]uint16, length)
		for i := range units {
			units[i] = binary.BigEndian.Uint16(data[pos+i*2:])
		}
		return string(utf16.Decode(units)), nil

	default:
		return "", fmt.Errorf("not a string object (type 0x%x)", typeTag)
	}
}

func encodeASCIIString(b []byte) []byte {
	return encodeStringPayload(0x5, b, len(b))
}

func encodeUnicodeString(s string) []byte {
	runes := []rune(s)
	units := utf16.Encode(runes)
	payload := make([]byte, len(units)*2)
	for i, u := range units {
		binary.BigEndian.PutUint16(payload[i*2:], u)
	}
	return encodeStringPayload(0x6, payload, len(units))
}

func encodeStringPayload(typeTag byte, payload []byte, length int) []byte {
	var header []byte
	if length < 15 {
		header = []byte{typeTag<<4 | byte(length)}
	} else {
		intBytes := intByteSize(uint64(length))
		header = make([]byte, 2+intBytes)
		header[0] = typeTag<<4 | 0x0f
		header[1] = 0x10 | intSizeNibble(intBytes)
		copy(header[2:], encodeUint(uint64(length), intBytes))
	}
	out := make([]byte, len(header)+len(payload))
	copy(out, header)
	copy(out[len(header):], payload)
	return out
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 127 {
			return false
		}
	}
	return true
}

// intByteSize returns the minimum number of bytes to encode v (1, 2, 4, or 8).
func intByteSize(v uint64) int {
	switch {
	case v <= 0xff:
		return 1
	case v <= 0xffff:
		return 2
	case v <= 0xffffffff:
		return 4
	default:
		return 8
	}
}

// intSizeNibble returns the power-of-two nibble for a byte size (0 for 1, 1 for 2, etc).
func intSizeNibble(byteSize int) byte {
	switch byteSize {
	case 1:
		return 0
	case 2:
		return 1
	case 4:
		return 2
	default:
		return 3
	}
}

func encodeUint(v uint64, size int) []byte {
	b := make([]byte, size)
	switch size {
	case 1:
		b[0] = byte(v)
	case 2:
		binary.BigEndian.PutUint16(b, uint16(v))
	case 4:
		binary.BigEndian.PutUint32(b, uint32(v))
	case 8:
		binary.BigEndian.PutUint64(b, v)
	}
	return b
}

func decodeUint(b []byte) uint64 {
	switch len(b) {
	case 1:
		return uint64(b[0])
	case 2:
		return uint64(binary.BigEndian.Uint16(b))
	case 4:
		return uint64(binary.BigEndian.Uint32(b))
	case 8:
		return binary.BigEndian.Uint64(b)
	default:
		return 0
	}
}
