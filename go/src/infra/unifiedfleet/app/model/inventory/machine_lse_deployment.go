// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package inventory

// MachineLSEDeploymentKind is the datastore entity kind for host deployment info.
const MachineLSEDeploymentKind string = "MachineLSEDeployment"

// MachineLSEDeploymentEntity is a datastore entity that tracks the deployment info for a host.
type MachineLSEDeploymentEntity struct {
	_kind                string `gae:"$kind,MachineLSEDeployment"`
	ID                   string `gae:"$id"`
	SerialNumber         string `gae:"serial_number"`
	DeploymentIdentifier string `gae:"deployment_identifier"`
	// Follow others entities, store ufspb.MachineLSEDeployment bytes.
	DeploymentInfo []byte `gae:",noindex"`
}
