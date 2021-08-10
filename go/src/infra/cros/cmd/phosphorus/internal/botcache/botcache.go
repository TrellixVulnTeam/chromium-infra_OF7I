// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package botcache provides an interface to interact with data cached in a
// swarming bot corresponding to a Chrome OS DUT.package botcache
package botcache

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"go.chromium.org/chromiumos/infra/proto/go/lab_platform"
	"go.chromium.org/luci/common/errors"
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
	if _, err := os.Stat(s.path()); os.IsNotExist(err) {
		// DUTs managed by a scheduling unit may not have their local state
		// file created until first test/admin task run, so we can just return
		// an empty DutState here.
		return &ds, nil
	}
	if err := readJSONPb(s.path(), &ds); err != nil {
		return nil, errors.Annotate(err, "load botcache for %s", s.Name).Err()
	}
	return &ds, nil
}

// Save overwrites the contents of the bot cache with provided DutState.
func (s *Store) Save(ds *lab_platform.DutState) error {
	if err := writeJSONPb(s.path(), ds); err != nil {
		return errors.Annotate(err, "save botcache for %s", s.Name).Err()
	}
	return nil
}

// LoadProvisionableLabel reads the value of the given provisionable label from
// the bot cache.
//
// LoadProvisionableLabel returns an empty string if the provided key does not
// exist in the provisionable labels in bot cache.
func (s *Store) LoadProvisionableLabel(name string) (string, error) {
	ds, err := s.Load()
	if err != nil {
		return "", err
	}
	for k, v := range ds.ProvisionableLabels {
		if k == name {
			return v, nil
		}
	}
	return "", nil
}

// SetNonEmptyProvisionableLabel sets the given provisionable label in the bot
// cache, provided the given value is not an empty string. Empty provisionable
// labels should never be set in the bot cache, and will result in an error.
//
// Use ClearProvisionableLabel to delete a label from cache.
func (s *Store) SetNonEmptyProvisionableLabel(name string, value string) error {
	if value == "" {
		return fmt.Errorf("attempted to set empty string for bot cache provisionable label %s", name)
	}
	ds, err := s.Load()
	if err != nil {
		return err
	}
	if ds.ProvisionableLabels == nil {
		ds.ProvisionableLabels = make(map[string]string)
	}
	ds.ProvisionableLabels[name] = value
	return s.Save(ds)
}

// ClearProvisionableLabel deletes a particular provisionable label from the
// bot cache.
func (s *Store) ClearProvisionableLabel(name string) error {
	ds, err := s.Load()
	if err != nil {
		return err
	}
	delete(ds.ProvisionableLabels, name)
	return s.Save(ds)
}

const (
	botCacheSubDir  = "swarming_state"
	botCacheFileExt = "json"
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
