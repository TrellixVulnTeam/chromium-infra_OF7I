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
	"strconv"
	"strings"

	"go.chromium.org/luci/common/errors"

	dirmdpb "infra/tools/dirmd/proto"
)

// The file implements reading of metadata from legacy OWNERS files.
// TODO(crbug.com/1104246): delete this file.

// OwnersFilename is the filename for OWNERS files.
const OwnersFilename = "OWNERS"

var ownerKeyValuePairRe = regexp.MustCompile(`#\s*([\w\-]+|Internal Component)\s*:\s*(\S+)`)

// ReadOwners reads metadata from legacy OWNERS of the given directory and
// returns a dirmdpb.Metadata message along with the lines that were ignored.
// Returns (nil, nil, nil) if OWNERS file does not exist.
func ReadOwners(dir string) (md *dirmdpb.Metadata, ignored []string, err error) {
	// Note: this function is case-sensitive wrt the filename,
	// because there are no OWNERS file in src.git that use different casing.

	f, err := os.Open(filepath.Join(dir, OwnersFilename))
	switch {
	case os.IsNotExist(err):
		return nil, nil, nil
	case err != nil:
		return nil, nil, err
	}
	defer f.Close()

	return parseOwners(f)
}

// parseOwners extracts metadata from a content of an OWNERS file.
func parseOwners(r io.Reader) (md *dirmdpb.Metadata, ignored []string, err error) {
	md = &dirmdpb.Metadata{}

	scan := bufio.NewScanner(r)
	for scan.Scan() {
		line := scan.Text()

		m := ownerKeyValuePairRe.FindStringSubmatch(line)
		if len(m) == 0 {
			ignored = append(ignored, line)
			continue
		}

		key, value := m[1], m[2]
		switch key {

		case "TEAM":
			md.TeamEmail = value

		case "COMPONENT":
			md.Monorail = &dirmdpb.Monorail{
				Project:   "chromium",
				Component: value,
			}

		case "OS":
			if md.Os, err = parseOSFromOwners(value); err != nil {
				return nil, nil, err
			}

		case "WPT-NOTIFY":
			switch strings.ToLower(value) {
			case "true":
				md.Wpt = &dirmdpb.WPT{Notify: dirmdpb.Trinary_YES}
			case "false":
				md.Wpt = &dirmdpb.WPT{Notify: dirmdpb.Trinary_NO}
			default:
				return nil, nil, errors.Reason("WPT-NOTIFY: expected true or false, got %q", value).Err()
			}

		case "Internal Component":
			const componentPrefix = "b/components/"
			if !strings.HasPrefix(value, componentPrefix) {
				return nil, nil, errors.Reason("Internal Component: expected component to start with 'b/components/', got %q", value).Err()
			}
			componentID, err := strconv.Atoi(strings.TrimPrefix(value, componentPrefix))
			if err != nil {
				return nil, nil, errors.Reason("Internal Component: expected integer component id, got %q", componentID).Err()
			}
			md.Buganizer = &dirmdpb.Buganizer{
				ComponentId: int64(componentID),
			}

		default:
			ignored = append(ignored, line)
		}
	}

	return md, ignored, nil
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
