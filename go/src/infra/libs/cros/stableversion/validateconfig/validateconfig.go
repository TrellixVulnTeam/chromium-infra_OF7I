// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package validateconfig

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"unicode/utf8"

	"github.com/golang/protobuf/jsonpb"

	sv "go.chromium.org/chromiumos/infra/proto/go/lab_platform"
)

var unmarshaller = jsonpb.Unmarshaler{AllowUnknownFields: false}

// FileNotReadable indicates that a given file does not exist or is not readable.
// For example, a directory is not a readable file.
const FileNotReadable = "File cannot be read (%s)"

// FileLenZero indicates that a file has length zero. This exceptional condition is special-cased because
// many weird files like /dev/null have length zero.
const FileLenZero = "File unexpectedly has length zero"

// FileNotUTF8 indicates that a file was found but is not valid UTF-8 and therefore not valid JSON either.
const FileNotUTF8 = "File is not valid UTF-8"

// FileNotJSON indicates that a file is not valid JSON.
const FileNotJSON = "File is not valid JSON"

// FileNotStableVersionProto indicates that a file does not conform to the lab_station.stable_versions proto schema.
// Not every file satisfying the schema is well-formed, but all files failing to satisfy the schema are definitely ill-formed.
const FileNotStableVersionProto = "File does not conform to stable_version proto"

// FileSeemsLegit indicates that that the given file is not malformed in any of the ways
// that the tool checks.
const FileSeemsLegit = "File appears to be a valid stable version config file."

// InspectFile takes a path and determines what, if anything, is wrong with a stable_versions.cfg file.
func InspectFile(path string) error {
	buf, err := ioutil.ReadFile(path)
	// TODO(gregorynisbet): finer grained errors for when you don't have permission to read a file or for when
	// the path is a directory
	if err != nil {
		return fmt.Errorf(FileNotReadable, err.Error())
	}
	return InspectBuffer(buf)
}

// InspectBuffer takes file contents and determines what, if anything, is wrong with a stable_versions.cfg file.
func InspectBuffer(contents []byte) error {
	if len(contents) == 0 {
		return fmt.Errorf(FileLenZero)
	}
	if !utf8.ValidString(string(contents)) {
		return fmt.Errorf(FileNotUTF8)
	}
	if !isValidJSON(contents) {
		return fmt.Errorf(FileNotJSON)
	}
	if _, err := parseStableVersions(contents); err != nil {
		return fmt.Errorf(FileNotStableVersionProto)
	}
	return nil
}

// isValidJSON determines whether a byte array contains valid JSON or not.
// This is used to give informative error messages if a non-JSON file is
// passed to ./stable_version2 validate-config .
func isValidJSON(contents []byte) bool {
	var sink json.RawMessage
	if err := json.Unmarshal(contents, &sink); err != nil {
		return false
	}
	return true
}

// parseStableVersions takes a byte array and attempts to parse a stable version
// proto file out of it.
func parseStableVersions(contents []byte) (*sv.StableVersions, error) {
	var allSV sv.StableVersions
	if err := unmarshaller.Unmarshal(bytes.NewReader(contents), &allSV); err != nil {
		return nil, err
	}
	return &allSV, nil
}
