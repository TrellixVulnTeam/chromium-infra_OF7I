// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package localinfo

import (
	"encoding/json"

	"go.chromium.org/luci/common/errors"
)

const (
	// CrosVersionKey is the cros version key in the ProvisionableLabels map.
	CrosVersionKey = "cros-version"
	// JobRepoURLKey is the job repo url key in the ProvisionableAttributes map.
	JobRepoURLKey = "job_repo_url"
)

// localDUTState contains persistent DUT information that is cached on the
// Skylab drone.
type localDUTState struct {
	ProvisionableLabels     provisionableLabels     `json:"provisionable_labels"`
	ProvisionableAttributes provisionableAttributes `json:"provisionable_attributes"`
}

// provisionableAttributes stores provisionable labels for a DUT.
type provisionableLabels map[string]string

// provisionableAttributes stores provisionable attributes for a DUT.
type provisionableAttributes map[string]string

// marshal returns the encoding of the localDUTState.
func (lds *localDUTState) marshal() ([]byte, error) {
	data, err := json.Marshal(lds)
	return data, errors.Annotate(err, "marshal").Err()
}

// unmarshal decodes localDUTState from the encoded data.
func (lds *localDUTState) unmarshal(data []byte) error {
	if err := json.Unmarshal(data, lds); err != nil {
		return err
	}
	if lds.ProvisionableLabels == nil {
		lds.ProvisionableLabels = make(map[string]string)
	}
	if lds.ProvisionableAttributes == nil {
		lds.ProvisionableAttributes = make(map[string]string)
	}
	return nil
}
