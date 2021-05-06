// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package inventory

import (
	"context"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/gae/service/datastore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ufspb "infra/unifiedfleet/api/v1/models"
	ufsds "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/util"
)

// MachineLSEDeploymentKind is the datastore entity kind for host deployment info.
const MachineLSEDeploymentKind string = "MachineLSEDeployment"

// MachineLSEDeploymentEntity is a datastore entity that tracks the deployment info for a host.
type MachineLSEDeploymentEntity struct {
	_kind                string `gae:"$kind,MachineLSEDeployment"`
	ID                   string `gae:"$id"`
	Hostname             string `gae:"hostname"`
	DeploymentIdentifier string `gae:"deployment_identifier"`
	DeploymentEnv        string `gae:"deployment_env"`
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
		DeploymentEnv:        p.GetDeploymentEnv().String(),
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

func getMachineLSEDeploymentID(pm proto.Message) string {
	p := pm.(*ufspb.MachineLSEDeployment)
	return p.GetSerialNumber()
}

// BatchGetMachineLSEDeployments returns a batch of deployment records from datastore.
func BatchGetMachineLSEDeployments(ctx context.Context, ids []string) ([]*ufspb.MachineLSEDeployment, error) {
	protos := make([]proto.Message, len(ids))
	for i, n := range ids {
		protos[i] = &ufspb.MachineLSEDeployment{SerialNumber: n}
	}
	pms, err := ufsds.BatchGet(ctx, protos, newMachineLSEDeploymentEntity, getMachineLSEDeploymentID)
	if err != nil {
		return nil, err
	}
	res := make([]*ufspb.MachineLSEDeployment, len(pms))
	for i, pm := range pms {
		res[i] = pm.(*ufspb.MachineLSEDeployment)
	}
	return res, nil
}

// ListMachineLSEDeployments lists the deployment records
//
// Does a query over MachineLSEDeploymentEntity. Returns up to pageSize entities, plus non-nil cursor (if
// there are more results). pageSize must be positive.
func ListMachineLSEDeployments(ctx context.Context, pageSize int32, pageToken string, filterMap map[string][]interface{}, keysOnly bool) (res []*ufspb.MachineLSEDeployment, nextPageToken string, err error) {
	q, err := ufsds.ListQuery(ctx, MachineLSEDeploymentKind, pageSize, pageToken, filterMap, keysOnly)
	if err != nil {
		return nil, "", err
	}
	var nextCur datastore.Cursor
	err = datastore.Run(ctx, q, func(ent *MachineLSEDeploymentEntity, cb datastore.CursorCB) error {
		if keysOnly {
			dr := &ufspb.MachineLSEDeployment{
				SerialNumber: ent.ID,
			}
			res = append(res, dr)
		} else {
			pm, err := ent.GetProto()
			if err != nil {
				logging.Errorf(ctx, "Failed to UnMarshal: %s", err)
				return nil
			}
			res = append(res, pm.(*ufspb.MachineLSEDeployment))
		}
		if len(res) >= int(pageSize) {
			if nextCur, err = cb(); err != nil {
				return err
			}
			return datastore.Stop
		}
		return nil
	})
	if err != nil {
		logging.Errorf(ctx, "Failed to list deployment records %s", err)
		return nil, "", status.Errorf(codes.Internal, ufsds.InternalError)
	}
	if nextCur != nil {
		nextPageToken = nextCur.String()
	}
	return
}

// DeleteDeployment deletes a deployment record
func DeleteDeployment(ctx context.Context, id string) error {
	return ufsds.Delete(ctx, &ufspb.MachineLSEDeployment{SerialNumber: id}, newMachineLSEDeploymentEntity)
}

// GetDeploymentIndexedFieldName returns the index name
func GetDeploymentIndexedFieldName(input string) (string, error) {
	var field string
	input = strings.TrimSpace(input)
	switch strings.ToLower(input) {
	case util.HostFilterName:
		field = "hostname"
	case util.DeploymentIdentifierFilterName:
		field = "deployment_identifier"
	default:
		return "", status.Errorf(codes.InvalidArgument, "Invalid field name %s - field name for machine lse deployments are host/deploymentidentifier", input)
	}
	return field, nil
}
