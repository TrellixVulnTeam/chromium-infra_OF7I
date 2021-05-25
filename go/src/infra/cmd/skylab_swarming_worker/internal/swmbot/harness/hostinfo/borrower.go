// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package hostinfo

import (
	"context"
	"fmt"
	"strings"

	"infra/libs/skylab/autotest/hostinfo"

	"infra/cmd/skylab_swarming_worker/internal/swmbot"
)

// Borrower represents borrowing LocalDUTState into a HostInfo.  It is used
// for returning any relevant Hostinfo changes back to the LocalDUTState.
type Borrower struct {
	hostInfo      *hostinfo.HostInfo
	localDUTState *swmbot.LocalDUTState
}

// BorrowLocalDUTState takes some info stored in the LocalDUTState and adds it to
// the HostInfo. The returned Borrower should be closed to return any
// relevant HostInfo changes back to the LocalDUTState.
func BorrowLocalDUTState(hi *hostinfo.HostInfo, lds *swmbot.LocalDUTState) *Borrower {
	for label, value := range lds.ProvisionableLabels {
		hi.Labels = append(hi.Labels, fmt.Sprintf("%s:%s", label, value))
	}
	for attribute, value := range lds.ProvisionableAttributes {
		hi.Attributes[attribute] = value
	}
	return &Borrower{
		hostInfo:      hi,
		localDUTState: lds,
	}
}

// Close returns any relevant Hostinfo changes back to the localDUTState.
// Subsequent calls do nothing. This is safe to call on a nil pointer.
func (b *Borrower) Close(ctx context.Context) error {
	if b == nil {
		return nil
	}
	if b.localDUTState == nil {
		return nil
	}
	hi, lds := b.hostInfo, b.localDUTState

	// copy provisioning labels
	for labelKey := range provisionableLabelKeys {
		delete(lds.ProvisionableLabels, labelKey)
	}
	for _, label := range hi.Labels {
		parts := strings.SplitN(label, ":", 2)
		if len(parts) != 2 {
			continue
		}
		if _, ok := provisionableLabelKeys[parts[0]]; ok {
			lds.ProvisionableLabels[parts[0]] = parts[1]
		}
	}

	// copy provisioning attributes
	for attrKey := range provisionableAttributeKeys {
		delete(lds.ProvisionableAttributes, attrKey)
	}
	for attribute, value := range hi.Attributes {
		if _, ok := provisionableAttributeKeys[attribute]; ok {
			lds.ProvisionableAttributes[attribute] = value
		}
	}
	b.localDUTState = nil
	return nil
}

var provisionableLabelKeys = map[string]struct{}{
	"cros-version": {},
	"fwro-version": {},
	"fwrw-version": {},
}

var provisionableAttributeKeys = map[string]struct{}{
	"job_repo_url": {},
	// Used to cache away changes to RPM power outlet state.
	"outlet_changed": {},
}
