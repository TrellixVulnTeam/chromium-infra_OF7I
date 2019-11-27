// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"os/exec"

	"infra/cmd/cros/thin-tls/api"
)

func (s *server) SetRpm(req *api.SetRpmRequest) (*api.SetRpmResponse, error) {
	var state string
	switch req.State {
	case api.SetRpmRequest_STATE_ON:
		state = "ON"
	case api.SetRpmRequest_STATE_OFF:
		state = "OFF"
	case api.SetRpmRequest_STATE_CYCLE:
		state = "CYCLE"
	default:
		return &api.SetRpmResponse{
			Status: api.SetRpmResponse_STATUS_BAD_REQUEST,
		}, nil
	}
	cmd := exec.Command("rpm_client",
		"-m", s.config.RPMMachine,
		"-s", state,
		"-p", s.config.PowerUnitHostname,
		"-o", s.config.PowerOutlet,
		"-y", s.config.HydraHostname)
	if err := cmd.Run(); err != nil {
		return &api.SetRpmResponse{
			Status:      api.SetRpmResponse_STATUS_UNKNOWN,
			Explanation: err.Error(),
		}, nil
	}
	return &api.SetRpmResponse{
		Status: api.SetRpmResponse_STATUS_OK,
	}, nil
}
