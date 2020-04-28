// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/common/errors"
)

// forEachFile calls f() once per file path specified via a list of glob
// patterns.
func forEachFile(globs []string, f func(string)) error {
	for _, g := range globs {
		ps, err := filepath.Glob(g)
		if err != nil {
			return errors.Annotate(err, "for each file").Err()
		}
		for _, p := range ps {
			f(p)
		}
	}
	return nil
}

func loadFromBinary(path string, m proto.Message) error {
	s, err := ioutil.ReadFile(path)
	if err != nil {
		return errors.Annotate(err, "load binary proto from %s", path).Err()
	}
	if err := proto.Unmarshal(s, m); err != nil {
		return errors.Annotate(err, "load binary proto from %s", path).Err()
	}
	return nil
}

var unmarshaller = jsonpb.Unmarshaler{AllowUnknownFields: true}

func loadFromJSON(path string, m proto.Message) error {
	r, err := os.Open(path)
	if err != nil {
		return errors.Annotate(err, "load JSON proto from %s", path).Err()
	}
	if err := unmarshaller.Unmarshal(r, m); err != nil {
		return errors.Annotate(err, "load JSON proto from %s", path).Err()
	}
	return nil
}
