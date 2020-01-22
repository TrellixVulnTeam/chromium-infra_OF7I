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

	labPlatform "go.chromium.org/chromiumos/infra/proto/go/lab_platform"
	"infra/libs/cros/stableversion"
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

// FileJSONNull indicates that the file is a null value in JSON
const FileJSONNull = "File is null JSON literal"

// FileNotStableVersionProto indicates that a file does not conform to the lab_station.stable_versions proto schema.
// Not every file satisfying the schema is well-formed, but all files failing to satisfy the schema are definitely ill-formed.
const FileNotStableVersionProto = "File does not conform to stable_version proto"

// FileMissingCrosKey indicates that a file does not have a "cros" key
const FileMissingCrosKey = "File is missing \"cros\" key"

// FileMissingFirmwareKey indicates that a file does not have a "firmware" key
const FileMissingFirmwareKey = "File is missing \"firmware\" key"

// FileMissingFaftKey indicates that a file does not have a "faft" key
const FileMissingFaftKey = "File is missing \"faft\" key"

// FileNoCrosEntries indicates that the "cros" array is empty.
const FileNoCrosEntries = "File has no \"cros\" entries"

// FileNoFirmwareEntries indicates that the "firmware" array is empty.
const FileNoFirmwareEntries = "File has no \"firmware\" entries"

// FileNoFaftEntries indicates that the "faft" array is empty.
const FileNoFaftEntries = "File has no \"faft\" entries"

// FileShallowlyMalformedCrosEntry indicates that there is at least one CrOS entry with a "shallow" error such as a
// malformed version string.
// Free variables: message string, position int, buildTarget string, model string, version string
const FileShallowlyMalformedCrosEntry = "File has bad CrOS version entry position (%s): (%d) buildTarget: (%s) model:(%s) version: (%s)"

func makeShallowlyMalformedCrosEntry(message string, index int, buildTarget string, model string, version string) error {
	return fmt.Errorf(FileShallowlyMalformedCrosEntry, message, index, buildTarget, model, version)
}

// FileShallowlyMalformedFirmwareEntry indicates that there is at least one CrOS entry with a "shallow" error such as a
// malformed version string.
// Free variables: message string, position int, buildTarget string, model string, version string
const FileShallowlyMalformedFirmwareEntry = "File has bad firmware version entry position (%s): (%d) buildTarget: (%s) model:(%s) version: (%s)"

func makeShallowlyMalformedFirmwareEntry(message string, index int, buildTarget string, model string, version string) error {
	return fmt.Errorf(FileShallowlyMalformedFirmwareEntry, message, index, buildTarget, model, version)
}

// FileShallowlyMalformedFaftEntry indicates that there is at least one CrOS entry with a "shallow" error such as a
// malformed version string.
// Free variables: message string, position int, buildTarget string, model string, version string
const FileShallowlyMalformedFaftEntry = "File has bad faft version entry position (%s): (%d) buildTarget: (%s) model:(%s) version: (%s)"

func makeShallowlyMalformedFaftEntry(message string, index int, buildTarget string, model string, version string) error {
	return fmt.Errorf(FileShallowlyMalformedFaftEntry, message, index, buildTarget, model, version)
}

// FileSeemsLegit indicates that that the given file is not malformed in any of the ways
// that the tool checks.
const FileSeemsLegit = "File appears to be a valid stable version config file."

// InspectFile takes a path and determines what, if anything, is wrong with a stable_versions.cfg file.
func InspectFile(path string) (*labPlatform.StableVersions, error) {
	buf, err := ioutil.ReadFile(path)
	// TODO(gregorynisbet): finer grained errors for when you don't have permission to read a file or for when
	// the path is a directory
	if err != nil {
		return nil, fmt.Errorf(FileNotReadable, err.Error())
	}
	return InspectBuffer(buf)
}

// InspectBuffer takes file contents and determines what, if anything, is wrong with a stable_versions.cfg file.
func InspectBuffer(contents []byte) (*labPlatform.StableVersions, error) {
	if len(contents) == 0 {
		return nil, fmt.Errorf(FileLenZero)
	}
	if !utf8.ValidString(string(contents)) {
		return nil, fmt.Errorf(FileNotUTF8)
	}
	if !isValidJSON(contents) {
		return nil, fmt.Errorf(FileNotJSON)
	}
	sv, err := ParseStableVersions(contents)
	if err != nil {
		return nil, err
	}
	return sv, validateStableVersions(sv)
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

// ParseStableVersions takes a byte array and attempts to parse a stable version
// proto file out of it.
func ParseStableVersions(contents []byte) (*labPlatform.StableVersions, error) {
	var allSV labPlatform.StableVersions
	if err := unmarshaller.Unmarshal(bytes.NewReader(contents), &allSV); err != nil {
		return nil, fmt.Errorf("JSON does not conform to schema: %s", err.Error())
	}
	return &allSV, nil
}

func validateStableVersions(sv *labPlatform.StableVersions) error {
	if sv == nil {
		return fmt.Errorf(FileJSONNull)
	}
	if sv.Cros == nil {
		return fmt.Errorf(FileMissingCrosKey)
	}
	if sv.Firmware == nil {
		return fmt.Errorf(FileMissingFirmwareKey)
	}
	if sv.Faft == nil {
		return fmt.Errorf(FileMissingFaftKey)
	}
	if len(sv.Cros) == 0 {
		return fmt.Errorf(FileNoCrosEntries)
	}
	if len(sv.Firmware) == 0 {
		return fmt.Errorf(FileNoFirmwareEntries)
	}
	if err := shallowValidateCrosVersions(sv); err != nil {
		return err
	}
	return nil
}

// shallowValidateCrosVersions checks the CrOS version entries and confirms
// that each entry is well-formed and that there are no duplicates.
func shallowValidateCrosVersions(sv *labPlatform.StableVersions) error {
	seen := make(map[string]bool)
	for i, cros := range sv.GetCros() {
		bt := cros.GetKey().GetBuildTarget().GetName()
		model := cros.GetKey().GetModelId().GetValue()
		version := cros.GetVersion()
		if bt == "" {
			return makeShallowlyMalformedCrosEntry("empty buildTarget", i, bt, model, version)
		}
		if model != "" {
			return makeShallowlyMalformedCrosEntry("non-empty models are NOT supported in CrOS versions", i, bt, model, version)
		}
		if seen[bt] {
			return makeShallowlyMalformedCrosEntry("duplicate entry for buildTarget", i, bt, model, version)
		}
		seen[bt] = true
		// model is explicitly allowed to be "" for a CrOS version.
		if err := stableversion.ValidateCrOSVersion(version); err != nil {
			return makeShallowlyMalformedCrosEntry("invalid version", i, bt, model, version)
		}
	}
	return nil
}

// shallowValidateFirmwareVersions checks the firmware version entries and confirms
// that each entry is well-formed and that there are no duplicates.
func shallowValidateFirmwareVersions(sv *labPlatform.StableVersions) error {
	seen := make(map[string]bool)
	for i, fw := range sv.GetFirmware() {
		bt := fw.GetKey().GetBuildTarget().GetName()
		model := fw.GetKey().GetModelId().GetValue()
		joined, err := stableversion.JoinBuildTargetModel(bt, model)
		if err != nil {
			return fmt.Errorf("shallowValidateFirmwareVersions: internal error: %s", err)
		}
		version := fw.GetVersion()
		if bt == "" {
			return makeShallowlyMalformedFirmwareEntry("empty buildTarget", i, bt, model, version)
		}
		if model == "" {
			return makeShallowlyMalformedFirmwareEntry("empty model", i, bt, model, version)
		}
		if seen[joined] {
			return makeShallowlyMalformedFirmwareEntry("duplicate entry", i, bt, model, version)
		}
		seen[joined] = true
		if err := stableversion.ValidateFirmwareVersion(version); err != nil {
			return makeShallowlyMalformedFirmwareEntry("invalid version", i, bt, model, version)
		}
	}
	return nil
}

// shallowValidateFaftVersions checks the firmware version entries and confirms
// that each entry is well-formed and that there are no duplicates.
func shallowValidateFaftVersions(sv *labPlatform.StableVersions) error {
	seen := make(map[string]bool)
	for i, fa := range sv.GetFaft() {
		bt := fa.GetKey().GetBuildTarget().GetName()
		model := fa.GetKey().GetModelId().GetValue()
		joined, err := stableversion.JoinBuildTargetModel(bt, model)
		if err != nil {
			return fmt.Errorf("shallowValidateFaftVersions: internal error: %s", err)
		}
		version := fa.GetVersion()
		if bt == "" {
			return makeShallowlyMalformedFaftEntry("empty buildTarget", i, bt, model, version)
		}
		if model == "" {
			return makeShallowlyMalformedFaftEntry("empty model", i, bt, model, version)
		}
		if seen[joined] {
			return makeShallowlyMalformedFaftEntry("duplicate entry", i, bt, model, version)
		}
		seen[joined] = true
		if err := stableversion.ValidateFaftVersion(version); err != nil {
			return makeShallowlyMalformedFaftEntry("invalid version", i, bt, model, version)
		}
	}
	return nil
}
