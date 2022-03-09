// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package recovery

import (
	"bytes"
	"encoding/json"
	"infra/cros/recovery/internal/planpb"
	"io"
	"log"
)

// createConfiguration creates configuration plan based on provided plan data.
func createConfiguration(plans []*planpb.Plan) *planpb.Configuration {
	if len(plans) == 0 {
		return nil
	}
	c := &planpb.Configuration{Plans: make(map[string]*planpb.Plan)}
	for _, p := range plans {
		c.PlanNames = append(c.PlanNames, p.GetName())
		c.Plans[p.GetName()] = p
	}
	return c
}

func createConfigurationJSON(plans []*planpb.Plan) ([]byte, error) {
	c := createConfiguration(plans)
	if c == nil {
		// Backwards compatibility.
		return []byte(""), nil
	}
	return json.Marshal(c)
}

func mustCreateConfigratuionJSON(plans []*planpb.Plan) io.Reader {
	c, err := createConfigurationJSON(plans)
	if err != nil {
		log.Fatalf("Failed to create repair configs: %v", err)
	}
	return bytes.NewBuffer(c)
}
