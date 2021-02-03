// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package botcache provides an interface to interact with data cached in a
// swarming bot corresponding to a Chrome OS DUT.package botcache
package botcache

import (
	"fmt"
	"path/filepath"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"go.chromium.org/chromiumos/infra/proto/go/lab_platform"
	"go.chromium.org/luci/common/errors"

	"os"
)

// Store exposes methods to read / write the contents of the bot cache.
type Store struct {
	// The top-level bot cache directory.
	// Usually the autotest results directory.
	CacheDir string
	// Bot specific identity.
	// Usually the DUT name (deprecated: Dut inventory ID).
	Name string
}

// Load reads the contents of the bot cache.
func (s *Store) Load() (*lab_platform.DutState, error) {
	ds := lab_platform.DutState{}
	if err := readJSONPb(s.path(), &ds); err != nil {
		return nil, errors.Annotate(err, "load botcache").Err()
	}
	return &ds, nil
}

// Save overwrites the contents of the bot cache with provided DutState.
func (s *Store) Save(ds *lab_platform.DutState) error {
	if err := writeJSONPb(s.path(), ds); err != nil {
		return errors.Annotate(err, "save botcache").Err()
	}
	return nil
}

const (
	botCacheSubDir  = "swarming_state"
	botCacheFileExt = ".json"
)

func (s *Store) path() string {
	return filepath.Join(s.CacheDir, botCacheSubDir, fmt.Sprintf("%s.%s", s.Name, botCacheFileExt))
}

func readJSONPb(inFile string, payload proto.Message) error {
	r, err := os.Open(inFile)
	if err != nil {
		return errors.Annotate(err, "read JSON pb").Err()
	}
	defer r.Close()

	unmarshaler := jsonpb.Unmarshaler{AllowUnknownFields: true}
	if err := unmarshaler.Unmarshal(r, payload); err != nil {
		return errors.Annotate(err, "read JSON pb").Err()
	}
	return nil
}

func writeJSONPb(outFile string, payload proto.Message) error {
	dir := filepath.Dir(outFile)
	// Create the directory if it doesn't exist.
	if err := os.MkdirAll(dir, 0777); err != nil {
		return errors.Annotate(err, "write JSON pb").Err()
	}

	w, err := os.Create(outFile)
	if err != nil {
		return errors.Annotate(err, "write JSON pb").Err()
	}
	defer w.Close()

	marshaler := jsonpb.Marshaler{}
	if err := marshaler.Marshal(w, payload); err != nil {
		return errors.Annotate(err, "write JSON pb").Err()
	}
	return nil
}
