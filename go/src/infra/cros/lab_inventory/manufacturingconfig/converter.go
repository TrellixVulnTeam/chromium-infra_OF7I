// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package manufacturingconfig

import (
	"go.chromium.org/chromiumos/infra/proto/go/manufacturing"
	"infra/libs/skylab/inventory"
)

// ConvertMCToV1Labels converts manufacturing configs to the git-based skylab inventory.
func ConvertMCToV1Labels(m *manufacturing.Config, l *inventory.SchedulableLabels) {
	if m == nil || l == nil {
		return
	}
	l.Phase = (*inventory.SchedulableLabels_Phase)(&(m.DevicePhase))
	l.Cr50Phase = (*inventory.SchedulableLabels_CR50_Phase)(&(m.Cr50Phase))
	cr50Env := ""
	switch m.Cr50KeyEnv {
	case manufacturing.Config_CR50_KEYENV_PROD:
		cr50Env = "prod"
	case manufacturing.Config_CR50_KEYENV_DEV:
		cr50Env = "dev"
	}
	if cr50Env != "" {
		l.Cr50RoKeyid = &cr50Env
	}
	if l.WifiChip == nil {
		l.WifiChip = new(string)
	}
	*l.WifiChip = m.GetWifiChip()
	l.HwidComponent = m.GetHwidComponent()
}
