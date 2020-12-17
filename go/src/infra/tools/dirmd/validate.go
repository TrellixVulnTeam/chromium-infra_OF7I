// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package dirmd

import (
	"io/ioutil"
	"os"

	"google.golang.org/protobuf/encoding/prototext"

	dirmdpb "infra/tools/dirmd/proto"
)

// ValidateFile returns a non-nil error if the metadata file is invalid.
//
// A valid file contents is Metadata protobuf-text message.
func ValidateFile(fileName string) error {
	f, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	contents, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	return prototext.Unmarshal(contents, &dirmdpb.Metadata{})
}
