// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package botinfo implements opening and closing a bot's botinfo stored on
// disk.
package botinfo

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"

	"go.chromium.org/luci/common/errors"

	"infra/cmd/skylab_swarming_worker/internal/swmbot"
	"infra/cros/dutstate"
)

// Store holds a bot's botinfo and dut name, and adds a Close method.
type Store struct {
	swmbot.LocalState
	bot *swmbot.Info
	// Ideally, dutName should be stored in swmbot.Info next to the DUTID
	// field, but the swmbot.Info fields are populated from env variables,
	// and dutName is fetched via an API call later on. So we store dutName
	// here to avoid carrying an uninitialized field around in swmbot.Info.
	dutName string
}

// Close writes the BotInfo back to disk.  This method does nothing on
// subsequent calls.  This method is safe to call on a nil pointer.
func (s *Store) Close(ctx context.Context) error {
	if s == nil {
		return nil
	}
	if s.bot == nil {
		return nil
	}
	data, err := swmbot.Marshal(&s.LocalState)
	if err != nil {
		return errors.Annotate(err, "close botinfo").Err()
	}
	// Write DUT state into two files: one by DUT name, one by DUT ID.
	// TODO(crbug.com/994404): Stop saving the DUT ID-based state file.
	if err := ioutil.WriteFile(botinfoFilePath(s.bot, s.bot.DUTID), data, 0666); err != nil {
		return errors.Annotate(err, "close botinfo").Err()
	}
	if err := ioutil.WriteFile(botinfoFilePath(s.bot, s.dutName), data, 0666); err != nil {
		return errors.Annotate(err, "close botinfo").Err()
	}
	ufsClient, err := swmbot.UFSClient(ctx, s.bot)
	if err != nil {
		return err
	}
	err = dutstate.Update(ctx, ufsClient, s.dutName, s.LocalState.HostState)
	if err != nil {
		return errors.Annotate(err, "close botinfo").Err()
	}
	s.bot = nil
	return nil
}

// Open loads the BotInfo for the Bot.  The BotInfo should be closed
// afterward to write it back.
func Open(ctx context.Context, b *swmbot.Info, dutName string) (*Store, error) {
	s := Store{bot: b, dutName: dutName}
	// Read provisioning info from state file.
	data, err := ioutil.ReadFile(botinfoFilePath(b, dutName))
	if err != nil {
		return nil, errors.Annotate(err, "open botinfo").Err()
	}
	if err := swmbot.Unmarshal(data, &s.LocalState); err != nil {
		return nil, errors.Annotate(err, "open botinfo").Err()
	}
	// Read DUT state from UFS.
	ufsClient, err := swmbot.UFSClient(ctx, b)
	if err != nil {
		return nil, errors.Annotate(err, "open botinfo").Err()
	}
	dutInfo := dutstate.Read(ctx, ufsClient, s.dutName)
	log.Printf("Received DUT state from UFS: %v", dutInfo)
	s.LocalState.HostState = dutInfo.State
	return &s, nil
}

// botinfoFilePath returns the path for caching dimensions for the given bot.
func botinfoFilePath(b *swmbot.Info, fileID string) string {
	return filepath.Join(botinfoDirPath(b), fmt.Sprintf("%s.json", fileID))
}

// botinfoDir returns the path to the cache directory for the given bot.
func botinfoDirPath(b *swmbot.Info) string {
	return filepath.Join(b.AutotestPath, "swarming_state")
}
