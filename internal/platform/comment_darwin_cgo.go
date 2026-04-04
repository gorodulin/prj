//go:build darwin && cgo

package platform

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation
#import <Foundation/Foundation.h>
#include <sys/xattr.h>
#include <stdlib.h>

// runAppleScript executes an AppleScript string.
// Returns 0 on success, 1 on error.
int runAppleScript(const char* script) {
	@autoreleasepool {
		NSString* src = [NSString stringWithUTF8String:script];
		NSAppleScript* as = [[NSAppleScript alloc] initWithSource:src];
		NSDictionary* error = nil;
		[as executeAndReturnError:&error];
		if (error) return 1;
		return 0;
	}
}
*/
import "C"

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

const finderCommentAttr = "com.apple.metadata:kMDItemFinderComment"

// FinderComment pairs a link path with its desired comment text.
type FinderComment struct {
	Path    string
	Comment string
}

// escapeAppleScriptString escapes a Go string for use inside AppleScript
// double-quoted string literals.
func escapeAppleScriptString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

// SetFinderComment sets the Finder comment on a single path.
func SetFinderComment(path, comment string) error {
	return SetFinderComments([]FinderComment{{Path: path, Comment: comment}})
}

// SetFinderComments sets Finder comments on multiple paths in a single batch.
// Uses one AppleScript execution for all items.
func SetFinderComments(items []FinderComment) error {
	if len(items) == 0 {
		return nil
	}

	var b strings.Builder
	b.WriteString("tell application \"Finder\"\n")
	for _, item := range items {
		dir := escapeAppleScriptString(filepath.Dir(item.Path))
		name := escapeAppleScriptString(filepath.Base(item.Path))
		comment := escapeAppleScriptString(item.Comment)
		fmt.Fprintf(&b,
			"set comment of item \"%s\" of (POSIX file \"%s\" as alias) to \"%s\"\n",
			name, dir, comment)
	}
	b.WriteString("end tell\n")

	cScript := C.CString(b.String())
	defer C.free(unsafe.Pointer(cScript))

	rc := C.runAppleScript(cScript)
	if rc != 0 {
		return fmt.Errorf("set finder comments: AppleScript failed (%d items)", len(items))
	}
	return nil
}

// GetFinderComment reads the Finder comment from path.
func GetFinderComment(path string) (string, error) {
	data, err := GetFinderCommentRaw(path)
	if err != nil {
		return "", err
	}
	if data == nil {
		return "", nil
	}
	return DecodeBplistString(data)
}

// GetFinderCommentRaw reads the raw xattr bytes of the Finder comment.
// For links (symlinks or aliases), reads from the target, matching Finder's behavior.
// Returns (nil, nil) if no comment is set.
func GetFinderCommentRaw(path string) ([]byte, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))
	cAttr := C.CString(finderCommentAttr)
	defer C.free(unsafe.Pointer(cAttr))

	// Query size. Options=0 follows links, matching where Finder stores comments.
	size, err := C.getxattr(cPath, cAttr, nil, 0, 0, 0)
	if err != nil {
		if err == syscall.ENOATTR {
			return nil, nil
		}
		return nil, fmt.Errorf("get finder comment size: %w", err)
	}
	if size == 0 {
		return nil, nil
	}

	buf := make([]byte, size)
	n, err := C.getxattr(cPath, cAttr, unsafe.Pointer(&buf[0]), C.size_t(size), 0, 0)
	if err != nil {
		return nil, fmt.Errorf("get finder comment: %w", err)
	}
	return buf[:n], nil
}

// FinderCommentChanged reports whether the encoded comment differs from
// what is currently stored on path. Used to skip unnecessary writes.
func FinderCommentChanged(path string, encoded []byte) (bool, error) {
	current, err := GetFinderCommentRaw(path)
	if err != nil {
		return true, err
	}
	return !bytes.Equal(current, encoded), nil
}
