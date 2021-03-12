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

	"go.chromium.org/luci/common/errors"

	dirmdpb "infra/tools/dirmd/proto"
)

// ValidateFile returns a non-nil error if the metadata file is invalid.
//
// A valid file has a base filename "DIR_METADATA" or "OWNERS".
// The format of its contents correspond to the base name.
func ValidateFile(fileName string) error {
	base := filepath.Base(fileName)
	if base != Filename && base != "OWNERS" {
		return errors.Reason("unexpected base filename %q; expected DIR_METADATA or OWNERS", base).Err()
	}

	f, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	if base == Filename {
		contents, err := ioutil.ReadAll(f)
		if err != nil {
			return err
		}
		md := &dirmdpb.Metadata{}
		if err := prototext.Unmarshal(contents, md); err != nil {
			return err
		}
		return Validate(md)
	}

	_, _, err = parseOwners(f)
	return err
}

// Validate returns a non-nil error if md is invalid.
func Validate(md *dirmdpb.Metadata) error {
	if err := validateInheritFrom(md.InheritFrom); err != nil {
		return errors.Annotate(err, "inherit_from").Err()
	}
	return nil
}

func validateInheritFrom(inheritFrom string) error {
	if strings.Contains(inheritFrom, "\\") {
		return errors.Reason("contains backslash; must use forward slash").Err()
	}
	if inheritFrom != "" && inheritFrom != "-" {
		// Must be a path.
		if !strings.HasPrefix(inheritFrom, "//") {
			return errors.Reason("must start with //").Err()
		}
	}
	return nil
}
