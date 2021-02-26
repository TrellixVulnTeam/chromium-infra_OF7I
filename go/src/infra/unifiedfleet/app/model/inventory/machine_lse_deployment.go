// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package inventory

import (
	"context"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"go.chromium.org/luci/common/errors"

	ufspb "infra/unifiedfleet/api/v1/models"
	ufsds "infra/unifiedfleet/app/model/datastore"
)

// MachineLSEDeploymentKind is the datastore entity kind for host deployment info.
const MachineLSEDeploymentKind string = "MachineLSEDeployment"

// MachineLSEDeploymentEntity is a datastore entity that tracks the deployment info for a host.
type MachineLSEDeploymentEntity struct {
	_kind                string `gae:"$kind,MachineLSEDeployment"`
	ID                   string `gae:"$id"`
	Hostname             string `gae:"hostname"`
	DeploymentIdentifier string `gae:"deployment_identifier"`
	// Follow others entities, store ufspb.MachineLSEDeployment bytes.
	DeploymentInfo []byte `gae:",noindex"`
}

// GetProto returns the unmarshaled MachineLSEDeploymentEntity.
func (e *MachineLSEDeploymentEntity) GetProto() (proto.Message, error) {
	var p ufspb.MachineLSEDeployment
	if err := proto.Unmarshal(e.DeploymentInfo, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func newMachineLSEDeploymentEntity(ctx context.Context, pm proto.Message) (ufsds.FleetEntity, error) {
	p := pm.(*ufspb.MachineLSEDeployment)
	if p.GetSerialNumber() == "" {
		return nil, errors.Reason("Empty MachineLSEDeployment serial number").Err()
	}
	deploymentInfo, err := proto.Marshal(p)
	if err != nil {
		return nil, errors.Annotate(err, "fail to marshal deployment info %s", p).Err()
	}
	return &MachineLSEDeploymentEntity{
		ID:                   p.GetSerialNumber(),
		Hostname:             p.GetHostname(),
		DeploymentIdentifier: p.GetDeploymentIdentifier(),
		DeploymentInfo:       deploymentInfo,
	}, nil
}

// UpdateMachineLSEDeployments updates the deployment infos for a batch of hosts in datastore.
func UpdateMachineLSEDeployments(ctx context.Context, dis []*ufspb.MachineLSEDeployment) ([]*ufspb.MachineLSEDeployment, error) {
	protos := make([]proto.Message, len(dis))
	utime := ptypes.TimestampNow()
	for i, di := range dis {
		di.UpdateTime = utime
		protos[i] = di
	}
	_, err := ufsds.PutAll(ctx, protos, newMachineLSEDeploymentEntity, true)
	if err == nil {
		return dis, err
	}
	return nil, err
}

// GetMachineLSEDeployment returns the deployment record for a given serial number
func GetMachineLSEDeployment(ctx context.Context, id string) (*ufspb.MachineLSEDeployment, error) {
	pm, err := ufsds.Get(ctx, &ufspb.MachineLSEDeployment{SerialNumber: id}, newMachineLSEDeploymentEntity)
	if err == nil {
		return pm.(*ufspb.MachineLSEDeployment), err
	}
	return nil, err
}

// DeleteDeployment deletes a deployment record
func DeleteDeployment(ctx context.Context, id string) error {
	return ufsds.Delete(ctx, &ufspb.MachineLSEDeployment{SerialNumber: id}, newMachineLSEDeploymentEntity)
}
