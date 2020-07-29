// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dutstate

import (
	"io/ioutil"
	"log"
	"path/filepath"

	"infra/cros/cmd/lucifer/internal/event"
)

const dutStateFilename = "dut_state.repair"

// ReadFile reads DUT state from dut_state.repair file and convert it to the event.
//
// The file will be exist if admin task requires to set special state to the DUT.
func ReadFile(resultsDir string) event.Event {
	if resultsDir == "" {
		return ""
	}

	path := filepath.Join(resultsDir, dutStateFilename)
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Println(err)
		return ""
	}
	state := string(data)
	log.Printf("The file %q contains DUT state: %q", path, state)
	return convertDUTStateToEvent(state)
}

// specialEvents represents list of expected special Events from admin tasks.
var specialEvents = map[event.Event]bool{
	event.HostNeedsManualRepair: true,
	event.HostNeedsReplacement:  true,
}

// convertDUTStateToEvent converts DUT state to the Event.
//
// The Event has to be present specialEvents set.
// The Event is a state with prefix 'host_'.
func convertDUTStateToEvent(state string) event.Event {
	if state == "" {
		return ""
	}
	e := event.Event("host_" + state)
	if _, ok := specialEvents[e]; ok {
		return e
	}
	log.Printf("unexpected DUT state: %q", state)
	return ""
}
