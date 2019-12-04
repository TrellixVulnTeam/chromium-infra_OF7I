// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package datastore

import (
	"fmt"
	"time"

	"go.chromium.org/gae/service/datastore"
)

// DeviceEntityID represents the ID of a device. We prefer use asset id as the id.
type DeviceEntityID string

// DeviceKind is the datastore entity kind for Device entities.
const DeviceKind string = "Device"

// DeviceEntity is a datastore entity that tracks a device.
type DeviceEntity struct {
	_kind     string         `gae:"$kind,Device"`
	ID        DeviceEntityID `gae:"$id"`
	Hostname  string
	LabConfig []byte `gae:",noindex"`
	Updated   time.Time
	Parent    *datastore.Key `gae:"$parent"`
}

func (e DeviceEntity) String() string {
	return fmt.Sprintf("<%s:%s>", e.Hostname, e.ID)
}
