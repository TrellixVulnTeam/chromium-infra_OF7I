// Copyright 2018 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package swmbot

import (
	"encoding/json"

	"infra/libs/cros/dutstate"
)

// LocalState contains persistent bot information that is cached on the
// Skylab drone.
type LocalState struct {
	HostState               dutstate.State          `json:"state"`
	ProvisionableLabels     ProvisionableLabels     `json:"provisionable_labels"`
	ProvisionableAttributes ProvisionableAttributes `json:"provisionable_attributes"`
}

// ProvisionableLabels stores provisionable labels for a bot and host.
type ProvisionableLabels map[string]string

// ProvisionableAttributes stores provisionable attributes for a bot and host.
type ProvisionableAttributes map[string]string

// Marshal returns the encoding of the BotInfo.
func Marshal(bi *LocalState) ([]byte, error) {
	return json.Marshal(bi)
}

// Unmarshal decodes BotInfo from the encoded data.
func Unmarshal(data []byte, bi *LocalState) error {
	if err := json.Unmarshal(data, bi); err != nil {
		return err
	}
	if bi.ProvisionableLabels == nil {
		bi.ProvisionableLabels = make(map[string]string)
	}
	if bi.ProvisionableAttributes == nil {
		bi.ProvisionableAttributes = make(map[string]string)
	}
	return nil
}
