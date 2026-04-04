//go:build darwin && cgo

package platform

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation
#import <Foundation/Foundation.h>
#include <stdlib.h>

// createFinderAlias creates a macOS Finder bookmark/alias file.
// Returns 0 on success, 1 if bookmark data creation failed, 2 if writing failed.
int createFinderAlias(const char* aliasPath, const char* targetPath) {
	@autoreleasepool {
		NSString* target = [NSString stringWithUTF8String:targetPath];
		NSString* alias = [NSString stringWithUTF8String:aliasPath];

		NSURL* targetURL = [NSURL fileURLWithPath:target];
		if (!targetURL) return 1;

		NSError* error = nil;
		NSData* bookmarkData = [targetURL
			bookmarkDataWithOptions:NSURLBookmarkCreationSuitableForBookmarkFile
			includingResourceValuesForKeys:nil
			relativeToURL:nil
			error:&error];
		if (!bookmarkData || error) return 1;

		NSURL* aliasURL = [NSURL fileURLWithPath:alias];
		BOOL ok = [NSURL writeBookmarkData:bookmarkData
			toURL:aliasURL
			options:NSURLBookmarkCreationSuitableForBookmarkFile
			error:&error];
		if (!ok || error) return 2;

		return 0;
	}
}

// resolveFinderAlias resolves a macOS Finder alias/bookmark to its target path.
// Writes the resolved path into out (up to outLen bytes).
// Returns 0 on success, 1 if the file is not an alias/bookmark, 2 on other error.
int resolveFinderAlias(const char* path, char* out, int outLen) {
	@autoreleasepool {
		NSString* filePath = [NSString stringWithUTF8String:path];
		NSURL* fileURL = [NSURL fileURLWithPath:filePath];
		if (!fileURL) return 2;

		NSError* error = nil;
		NSURL* resolved = [NSURL URLByResolvingAliasFileAtURL:fileURL
			options:NSURLBookmarkResolutionWithoutUI
			error:&error];

		if (!resolved || error) return 1;

		// Check that it actually resolved to something different.
		NSString* resolvedPath = [resolved path];
		if ([resolvedPath isEqualToString:filePath]) return 1;

		const char* utf8 = [resolvedPath UTF8String];
		if (!utf8) return 2;

		size_t len = strlen(utf8);
		if ((int)len >= outLen) return 2;

		memcpy(out, utf8, len + 1);
		return 0;
	}
}
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// CreateAlias creates a macOS Finder alias (bookmark file) at aliasPath
// pointing to targetPath. Both paths must be absolute.
func CreateAlias(aliasPath, targetPath string) error {
	cAlias := C.CString(aliasPath)
	defer C.free(unsafe.Pointer(cAlias))
	cTarget := C.CString(targetPath)
	defer C.free(unsafe.Pointer(cTarget))

	rc := C.createFinderAlias(cAlias, cTarget)
	switch rc {
	case 0:
		return nil
	case 1:
		return fmt.Errorf("create alias: failed to create bookmark data for %s", targetPath)
	case 2:
		return fmt.Errorf("create alias: failed to write bookmark file %s", aliasPath)
	default:
		return fmt.Errorf("create alias: unknown error (%d)", rc)
	}
}

// ResolveAlias resolves a macOS Finder alias or bookmark file to its target.
// Returns an error if the path is not an alias/bookmark.
func ResolveAlias(path string) (string, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	const maxPath = 4096
	out := (*C.char)(C.malloc(maxPath))
	defer C.free(unsafe.Pointer(out))

	rc := C.resolveFinderAlias(cPath, out, maxPath)
	switch rc {
	case 0:
		return C.GoString(out), nil
	case 1:
		return "", fmt.Errorf("not a finder alias: %s", path)
	default:
		return "", fmt.Errorf("resolve alias: failed for %s", path)
	}
}
