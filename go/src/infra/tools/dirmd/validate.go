// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package dirmd

import (
	"io/ioutil"
	"os"
	"path/filepath"

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
		return prototext.Unmarshal(contents, &dirmdpb.Metadata{})
	}

	_, _, err = parseOwners(f)
	return err
}
