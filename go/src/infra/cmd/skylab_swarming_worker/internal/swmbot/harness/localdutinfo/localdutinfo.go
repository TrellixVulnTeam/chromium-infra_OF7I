// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package localdutinfo implements opening and closing a DUT's local dut
// info stored on local disk(e.g. on skylab drones).
package localdutinfo

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"go.chromium.org/luci/common/errors"

	"infra/cmd/skylab_swarming_worker/internal/swmbot"
	"infra/cros/dutstate"
)

// Store holds a DUT's LocalDUTState and adds a Close method.
type Store struct {
	swmbot.LocalDUTState
	bot         *swmbot.Info
	dutHostname string
}

// Close writes the LocalDUTState back to disk.  This method does nothing on
// subsequent calls.  This method is safe to call on a nil pointer.
func (s *Store) Close(ctx context.Context) error {
	if s == nil {
		return nil
	}
	if s.bot == nil {
		return nil
	}
	data, err := swmbot.Marshal(&s.LocalDUTState)
	if err != nil {
		return errors.Annotate(err, "close localdutinfo").Err()
	}
	// Write DUT state into a local file named by DUT's hostname.
	if err := ioutil.WriteFile(localDUTInfoFilePath(s.bot, s.dutHostname), data, 0666); err != nil {
		return errors.Annotate(err, "close localdutinfo").Err()
	}
	ufsClient, err := swmbot.UFSClient(ctx, s.bot)
	if err != nil {
		return err
	}
	err = dutstate.Update(ctx, ufsClient, s.dutHostname, s.LocalDUTState.HostState)
	if err != nil {
		return errors.Annotate(err, "close localdutinfo").Err()
	}
	s.bot = nil
	return nil
}

// Open loads the LocalDUTInfo for the DUT. The LocalDUTInfo should be closed
// afterward to write it back.
func Open(ctx context.Context, b *swmbot.Info, dutHostname string) (*Store, error) {
	s := Store{bot: b, dutHostname: dutHostname}
	if err := s.retrieveLocalState(); err != nil {
		return nil, errors.Annotate(err, "retrieve local state").Err()
	}
	// Read DUT state from UFS.
	// TODO: Move remote dut state read to ufsDutInto.
	ufsClient, err := swmbot.UFSClient(ctx, b)
	if err != nil {
		return nil, errors.Annotate(err, "read DUT state from UFS").Err()
	}
	dutInfo := dutstate.Read(ctx, ufsClient, s.dutHostname)
	log.Printf("Received DUT state from UFS: %v", dutInfo)
	s.LocalDUTState.HostState = dutInfo.State
	return &s, nil
}

// localDUTInfoFilePath returns the path for caching dimensions for the given DUT.
func localDUTInfoFilePath(b *swmbot.Info, fileName string) string {
	return filepath.Join(localDUTInfoDirPath(b), fmt.Sprintf("%s.json", fileName))
}

// localDUTInfoDirPath returns the path to the cache directory for the given DUT.
func localDUTInfoDirPath(b *swmbot.Info) string {
	return filepath.Join(b.AutotestPath, "swarming_state")
}

// retrieveLocalState read DUT state data from local file and unmarshal the data.
// If the read fails due to target file not exists, the method will initialize
// an empty LocalDUTState.
func (s *Store) retrieveLocalState() error {
	data, err := ioutil.ReadFile(localDUTInfoFilePath(s.bot, s.dutHostname))
	if err != nil && !os.IsNotExist(err) {
		return errors.Annotate(err, "read local state file").Err()
	}
	if os.IsNotExist(err) {
		// If the file not exists we Marshal and Unmarshal a nil value here
		// to take advantage of Unmarshal initializing nested maps.
		data, err = swmbot.Marshal(nil)
		if err != nil {
			return errors.Annotate(err, "marshal nil local state").Err()
		}
	}
	if err := swmbot.Unmarshal(data, &s.LocalDUTState); err != nil {
		return errors.Annotate(err, "unmarshal data from state file ").Err()
	}
	return nil
}
