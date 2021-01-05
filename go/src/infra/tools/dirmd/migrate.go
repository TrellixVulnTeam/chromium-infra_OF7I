// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package dirmd

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"

	dirmdpb "infra/tools/dirmd/proto"
)

// The file implements migrating metadata from legacy OWNERS files into
// DIR_METADATA files.
// TODO(crbug.com/1104246): delete this file.

var emptyMD = &dirmdpb.Metadata{}
var emptyMonorail = &dirmdpb.Monorail{}

// MigrateMetadata moves metadata from legacy OWNERS of the given directory into
// DIR_METADATA file.
func MigrateMetadata(dir string) error {
	md, owners, err := ReadOwners(dir)
	if err != nil {
		return err
	}
	if md == nil || proto.Equal(md, emptyMD) {
		return nil
	}
	if err = writeMD(filepath.Join(dir, Filename), md); err != nil {
		return err
	}
	return writeOwners(filepath.Join(dir, OwnersFilename), owners)
}

func writeMD(path string, md *dirmdpb.Metadata) error {
	// Clear Monorail.Project field, since it's redundant for Chromium
	// DIR_METADATA files as "chromium" is set as project on the root.
	if md.Monorail != nil {
		md.Monorail.Project = ""
	}
	if proto.Equal(md.Monorail, emptyMonorail) {
		md.Monorail = nil
	}
	return ioutil.WriteFile(path, []byte(prototext.Format(md)), 0644)
}

// filterEmptyLines filters out duplicate empty lines, and empty lines at the
// beginning and end of the file.
func filterEmptyLines(lines []string) (filtered []string) {
	if lines == nil {
		return nil
	}
	lastLineEmpty := true
	filtered = make([]string, 0, len(lines))
	for _, line := range lines {
		// Skip duplicate empty lines
		if line == "" && lastLineEmpty {
			continue
		}
		lastLineEmpty = line == ""
		filtered = append(filtered, line)
	}
	// Make sure there is a newline at the end of file.
	if len(filtered) != 0 && filtered[len(filtered)-1] != "" {
		filtered = append(filtered, "")
	}
	return filtered
}

func writeOwners(path string, lines []string) error {
	// Filter empty lines from OWNERS files that might be left behind when
	// moving metadata out.
	lines = filterEmptyLines(lines)
	// Remove the resulting OWNERS file if all contents were moved.
	if len(lines) == 0 {
		return os.Remove(path)
	}
	return ioutil.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
}
