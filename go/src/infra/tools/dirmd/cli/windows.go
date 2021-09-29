// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// +build windows

package cli

import (
	"strings"

	"golang.org/x/sys/windows"

	"go.chromium.org/luci/common/errors"
)

// canonicalFSPath handles Windows-specific subst-ed drives.
// For example, if drive D: is mapped to C:\chromium\src and path is D:\foo, then
// C:\chromium\src\foo is returned.
func canonicalFSPath(path string) (targetPath string, err error) {
	// Use GetFinalPathNameByHandle to retrieve the canonical path.
	// It resolves subst-ed drives.

	// First create a file handle.
	// Docs: https://docs.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-createfilea
	handle, err := windows.CreateFile(
		windows.StringToUTF16Ptr(path),
		windows.GENERIC_READ,               // open for reading
		windows.FILE_SHARE_READ,            // share for reading
		nil,                                // default security
		windows.OPEN_EXISTING,              // existing file/dir only
		windows.FILE_FLAG_BACKUP_SEMANTICS, // use FILE_FLAG_BACKUP_SEMANTICS to open directories
		0)                                  // no template file
	if err != nil {
		return "", errors.Annotate(err, "failed to open %q", path).Err()
	}
	defer func() {
		closeErr := windows.CloseHandle(handle)
		if err == nil {
			err = closeErr
		}
	}()

	finalPathBuf := make([]uint16, 4086)
	// Docs: https://docs.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-getfinalpathnamebyhandlew
	n, err := windows.GetFinalPathNameByHandle(
		handle,
		&finalPathBuf[0],
		uint32(len(finalPathBuf)), // buffer size
		0,
	)
	switch {
	case err != nil:
		return "", err
	case int(n) > len(finalPathBuf):
		return "", errors.Annotate(err, "the final path is too long").Err()
	}
	finalPath := windows.UTF16PtrToString(&finalPathBuf[0])
	finalPath = strings.TrimPrefix(finalPath, `\\?\`)

	// Note: do not call CloseHandle twice, since the handle could be reused
	// immediately after closing.
	return finalPath, nil
}
