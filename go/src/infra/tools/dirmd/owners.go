// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package dirmd

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"go.chromium.org/luci/common/errors"

	dirmdpb "infra/tools/dirmd/proto"
)

// The file implements reading of metadata from legacy OWNERS files.
// TODO(crbug.com/1104246): delete this file.

var ownerKeyValuePairRe = regexp.MustCompile(`#\s*([\w\-]+)\s*:\s*(\S+)`)

// readOwners reads metadata from legacy OWNERS of the given directory.
// Returns (nil, nil) if OWNERS file does not exist.
func readOwners(dir string) (*dirmdpb.Metadata, error) {
	// Note: this function is case-sensitive wrt the filename,
	// because there are no OWNERS file in src.git that use different casing.

	f, err := os.Open(filepath.Join(dir, "OWNERS"))
	switch {
	case os.IsNotExist(err):
		return nil, nil
	case err != nil:
		return nil, err
	}
	defer f.Close()

	return parseOwners(f)
}

// parseOwners extracts metadata from a content of an OWNERS file.
func parseOwners(r io.Reader) (*dirmdpb.Metadata, error) {
	ret := &dirmdpb.Metadata{}

	scan := bufio.NewScanner(r)
	for scan.Scan() {
		line := scan.Text()
		if m := ownerKeyValuePairRe.FindStringSubmatch(line); len(m) > 0 {
			key, value := m[1], m[2]
			switch key {

			case "TEAM":
				ret.TeamEmail = value

			case "COMPONENT":
				ret.Monorail = &dirmdpb.Monorail{
					Project:   "chromium",
					Component: value,
				}

			case "OS":
				var err error
				if ret.Os, err = parseOSFromOwners(value); err != nil {
					return nil, err
				}

			case "WPT-NOTIFY":
				switch strings.ToLower(value) {
				case "true":
					ret.Wpt = &dirmdpb.WPT{Notify: dirmdpb.Trinary_YES}
				case "false":
					ret.Wpt = &dirmdpb.WPT{Notify: dirmdpb.Trinary_NO}
				default:
					return nil, errors.Reason("WPT-NOTIFY: expected true or false, got %q", value).Err()
				}
			}
		}
	}

	return ret, nil
}

// parseOSFromOwners parses a value of "OS" key in an OWNERS file
// to OS enum.
func parseOSFromOwners(s string) (dirmdpb.OS, error) {
	s = strings.ToUpper(s)

	if s == "CHROMEOS" {
		// ChromeOS is the only one for which the code below does not work.
		return dirmdpb.OS_CHROME_OS, nil
	}

	value := dirmdpb.OS(dirmdpb.OS_value[s])
	if value == dirmdpb.OS_OS_UNSPECIFIED {
		return 0, errors.Reason("failed to parse %q as an OS", s).Err()
	}

	return value, nil
}
