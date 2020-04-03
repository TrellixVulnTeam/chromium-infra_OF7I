// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package api

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Validate validates input requests of CreateMachines and UpdateMachines.
func (r *MachineList) Validate() error {
	if r.Machine == nil || len(r.Machine) == 0 {
		return status.Errorf(codes.InvalidArgument, "no Machine to add/update")
	}
	return nil
}
