// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package recovery

import (
	"infra/cros/recovery/internal/planpb"

	"google.golang.org/protobuf/types/known/durationpb"
)

var chameleonPlan = &planpb.Plan{
	CriticalActions: []string{
		"Mark as bad",
		"Device is pingable",
		"cros_ssh",
		"Mark as good",
	},
	Actions: map[string]*planpb.Action{
		"Mark as bad":  {ExecName: "chameleon_state_broken"},
		"Mark as good": {ExecName: "chameleon_state_working"},
		"Device is pingable": {
			ExecTimeout: &durationpb.Duration{Seconds: 15},
			ExecName:    "cros_ping",
		},
	},
}
