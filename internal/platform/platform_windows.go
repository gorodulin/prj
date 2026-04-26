//go:build windows

package platform

import (
	"encoding/binary"
	"fmt"
	"os"
	"syscall"
)

// IOCTL codes and reparse tags. Values from Microsoft's
// winioctl.h / ntifs.h headers; stable across all Windows versions.
const (
	fsctlSetReparsePoint     = 0x000900A4
	fsctlGetReparsePoint     = 0x000900A8
	ioReparseTagMountPoint   = 0xA0000003
	ioReparseTagSymlink      = 0xA000000C
	fileFlagOpenReparsePoint = 0x00200000
	fileFlagBackupSemantics  = 0x02000000

	// maxReparseBufferSize is the documented MAXIMUM_REPARSE_DATA_BUFFER_SIZE.
	maxReparseBufferSize = 16 * 1024
)

// ResolveLink resolves symlinks and NTFS junctions.
// Returns (target, kind, error) where kind is "symlink" or "junction".
func ResolveLink(path string) (string, string, error) {
	target, err := os.Readlink(path)
	if err != nil {
		return "", "", err
	}
	kind, kerr := reparseKind(path)
	if kerr != nil {
		// Tag read failed — fall back to symlink (the common case).
		// The user still gets a working target; only kind reporting is degraded.
		kind = "symlink"
	}
	return target, kind, nil
}

// SupportedLinkTypes returns link types available on Windows.
func SupportedLinkTypes() []string {
	return []string{"symlink", "junction"}
}

// DefaultLinkKind returns the link kind to use when none is configured.
// Junction is preferred on Windows because it requires no privileges,
// unlike symlinks which need Developer Mode or admin rights.
func DefaultLinkKind() string {
	return "junction"
}

// CreateJunction creates an NTFS directory junction at linkPath pointing to target.
// linkPath must not already exist; target must be an absolute path to an existing
// directory on the same volume. Junctions require no privileges.
func CreateJunction(linkPath, target string) error {
	if err := os.Mkdir(linkPath, 0755); err != nil {
		return fmt.Errorf("create junction dir: %w", err)
	}

	success := false
	defer func() {
		if !success {
			os.Remove(linkPath)
		}
	}()

	handle, err := openReparsePoint(linkPath, syscall.GENERIC_WRITE)
	if err != nil {
		return fmt.Errorf("open junction dir: %w", err)
	}
	defer syscall.CloseHandle(handle)

	buf, err := buildMountPointReparseBuffer(target)
	if err != nil {
		return err
	}

	var bytesReturned uint32
	if err := syscall.DeviceIoControl(
		handle,
		fsctlSetReparsePoint,
		&buf[0],
		uint32(len(buf)),
		nil,
		0,
		&bytesReturned,
		nil,
	); err != nil {
		return fmt.Errorf("set reparse point: %w", err)
	}

	success = true
	return nil
}

// reparseKind reads the reparse tag at path and returns "junction" or "symlink".
// Returns an error if path is not a reparse point or the tag is unrecognized.
func reparseKind(path string) (string, error) {
	handle, err := openReparsePoint(path, 0)
	if err != nil {
		return "", err
	}
	defer syscall.CloseHandle(handle)

	var buf [maxReparseBufferSize]byte
	var bytesReturned uint32
	if err := syscall.DeviceIoControl(
		handle,
		fsctlGetReparsePoint,
		nil,
		0,
		&buf[0],
		uint32(len(buf)),
		&bytesReturned,
		nil,
	); err != nil {
		return "", err
	}
	if bytesReturned < 4 {
		return "", fmt.Errorf("reparse buffer too small (%d bytes)", bytesReturned)
	}
	switch binary.LittleEndian.Uint32(buf[0:4]) {
	case ioReparseTagMountPoint:
		return "junction", nil
	case ioReparseTagSymlink:
		return "symlink", nil
	default:
		return "", fmt.Errorf("unknown reparse tag: 0x%08X", binary.LittleEndian.Uint32(buf[0:4]))
	}
}

// openReparsePoint opens path with FILE_FLAG_OPEN_REPARSE_POINT (don't follow)
// and FILE_FLAG_BACKUP_SEMANTICS (required for directories).
func openReparsePoint(path string, access uint32) (syscall.Handle, error) {
	pathUTF16, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	return syscall.CreateFile(
		pathUTF16,
		access,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE,
		nil,
		syscall.OPEN_EXISTING,
		fileFlagOpenReparsePoint|fileFlagBackupSemantics,
		0,
	)
}

// buildMountPointReparseBuffer constructs the REPARSE_DATA_BUFFER for a junction
// pointing to target. Layout (all little-endian):
//
//	ULONG  ReparseTag           = IO_REPARSE_TAG_MOUNT_POINT
//	USHORT ReparseDataLength
//	USHORT Reserved             = 0
//	USHORT SubstituteNameOffset = 0
//	USHORT SubstituteNameLength
//	USHORT PrintNameOffset      = SubstituteNameLength + 2 (skip null)
//	USHORT PrintNameLength
//	WCHAR  PathBuffer[]         = SubstituteName \0 PrintName \0
//
// SubstituteName is the NT path form (\??\ + absolute path); PrintName is the
// user-friendly form (just the absolute path).
func buildMountPointReparseBuffer(target string) ([]byte, error) {
	subst, err := utf16Bytes(`\??\` + target)
	if err != nil {
		return nil, err
	}
	print, err := utf16Bytes(target)
	if err != nil {
		return nil, err
	}

	pathBuf := make([]byte, 0, len(subst)+2+len(print)+2)
	pathBuf = append(pathBuf, subst...)
	pathBuf = append(pathBuf, 0, 0)
	pathBuf = append(pathBuf, print...)
	pathBuf = append(pathBuf, 0, 0)

	const headerAfterReserved = 8 // four USHORTs (offsets + lengths)
	dataLen := uint16(headerAfterReserved + len(pathBuf))

	buf := make([]byte, 8+int(dataLen))
	binary.LittleEndian.PutUint32(buf[0:4], ioReparseTagMountPoint)
	binary.LittleEndian.PutUint16(buf[4:6], dataLen)
	binary.LittleEndian.PutUint16(buf[6:8], 0) // Reserved
	binary.LittleEndian.PutUint16(buf[8:10], 0)
	binary.LittleEndian.PutUint16(buf[10:12], uint16(len(subst)))
	binary.LittleEndian.PutUint16(buf[12:14], uint16(len(subst)+2))
	binary.LittleEndian.PutUint16(buf[14:16], uint16(len(print)))
	copy(buf[16:], pathBuf)
	return buf, nil
}

// utf16Bytes encodes s as UTF-16 little-endian bytes, without a null terminator.
func utf16Bytes(s string) ([]byte, error) {
	u, err := syscall.UTF16FromString(s)
	if err != nil {
		return nil, err
	}
	// UTF16FromString appends a null; drop it.
	if len(u) > 0 && u[len(u)-1] == 0 {
		u = u[:len(u)-1]
	}
	out := make([]byte, len(u)*2)
	for i, c := range u {
		binary.LittleEndian.PutUint16(out[i*2:], c)
	}
	return out, nil
}
