// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package configuration

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

// VlanKind is the datastore entity kind Vlan.
const VlanKind string = "Vlan"

// VlanEntity is a datastore entity that tvlans Vlan.
type VlanEntity struct {
	_kind     string   `gae:"$kind,Vlan"`
	ID        string   `gae:"$id"`
	State     string   `gae:"state"`
	CidrBlock string   `gae:"cidr_block"`
	Zones     []string `gae:"zone"`
	Tags      []string `gae:"tags"`
	// ufspb.Vlan cannot be directly used as it contains pointer.
	Vlan []byte `gae:",noindex"`
}

// GetProto returns the unmarshaled Vlan.
func (e *VlanEntity) GetProto() (proto.Message, error) {
	var p ufspb.Vlan
	if err := proto.Unmarshal(e.Vlan, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func newVlanEntity(ctx context.Context, pm proto.Message) (ufsds.FleetEntity, error) {
	p := pm.(*ufspb.Vlan)
	if p.GetName() == "" {
		return nil, errors.Reason("Empty Vlan ID").Err()
	}
	vlan, err := proto.Marshal(p)
	if err != nil {
		return nil, errors.Annotate(err, "fail to marshal Vlan %s", p).Err()
	}
	zones := make([]string, len(p.GetZones()))
	for i, z := range p.GetZones() {
		zones[i] = z.String()
	}
	return &VlanEntity{
		ID:        p.GetName(),
		State:     p.GetResourceState().String(),
		CidrBlock: p.GetVlanAddress(),
		Zones:     zones,
		Tags:      p.GetTags(),
		Vlan:      vlan,
	}, nil
}

// CreateVlan creates a new vlan in datastore.
func CreateVlan(ctx context.Context, vlan *ufspb.Vlan) (*ufspb.Vlan, error) {
	return putVlan(ctx, vlan, false)
}

// UpdateVlan updates vlan in datastore.
func UpdateVlan(ctx context.Context, vlan *ufspb.Vlan) (*ufspb.Vlan, error) {
	return putVlan(ctx, vlan, true)
}

// BatchUpdateVlans updates a batch of vlans to datastore
//
// Can be used in a transaction
func BatchUpdateVlans(ctx context.Context, vlans []*ufspb.Vlan) ([]*ufspb.Vlan, error) {
	protos := make([]proto.Message, len(vlans))
	updateTime := ptypes.TimestampNow()
	for i, vlan := range vlans {
		vlan.UpdateTime = updateTime
		protos[i] = vlan
	}
	_, err := ufsds.PutAll(ctx, protos, newVlanEntity, true)
	if err == nil {
		return vlans, err
	}
	return nil, err
}

// GetVlan returns vlan for the given id from datastore.
func GetVlan(ctx context.Context, id string) (*ufspb.Vlan, error) {
	pm, err := ufsds.Get(ctx, &ufspb.Vlan{Name: id}, newVlanEntity)
	if err == nil {
		return pm.(*ufspb.Vlan), err
	}
	return nil, err
}

func getVlanID(pm proto.Message) string {
	p := pm.(*ufspb.Vlan)
	return p.GetName()
}

// BatchGetVlans returns a batch of vlans from datastore.
func BatchGetVlans(ctx context.Context, ids []string) ([]*ufspb.Vlan, error) {
	protos := make([]proto.Message, len(ids))
	for i, n := range ids {
		protos[i] = &ufspb.Vlan{Name: n}
	}
	pms, err := ufsds.BatchGet(ctx, protos, newVlanEntity, getVlanID)
	if err != nil {
		return nil, err
	}
	res := make([]*ufspb.Vlan, len(pms))
	for i, pm := range pms {
		res[i] = pm.(*ufspb.Vlan)
	}
	return res, nil
}

// ListVlans lists the vlans
//
// Does a query over Vlan entities. Returns up to pageSize entities, plus non-nil cursor (if
// there are more results). pageSize must be positive.
func ListVlans(ctx context.Context, pageSize int32, pageToken string, filterMap map[string][]interface{}, keysOnly bool) (res []*ufspb.Vlan, nextPageToken string, err error) {
	q, err := ufsds.ListQuery(ctx, VlanKind, pageSize, pageToken, filterMap, keysOnly)
	if err != nil {
		return nil, "", err
	}
	var nextCur datastore.Cursor
	err = datastore.Run(ctx, q, func(ent *VlanEntity, cb datastore.CursorCB) error {
		if keysOnly {
			vlan := &ufspb.Vlan{
				Name: ent.ID,
			}
			res = append(res, vlan)
		} else {
			pm, err := ent.GetProto()
			if err != nil {
				logging.Errorf(ctx, "Failed to UnMarshal: %s", err)
				return nil
			}
			res = append(res, pm.(*ufspb.Vlan))
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
		logging.Errorf(ctx, "Failed to List Vlans %s", err)
		return nil, "", status.Errorf(codes.Internal, ufsds.InternalError)
	}
	if nextCur != nil {
		nextPageToken = nextCur.String()
	}
	return
}

// DeleteVlan deletes the vlan in datastore
func DeleteVlan(ctx context.Context, id string) error {
	return ufsds.Delete(ctx, &ufspb.Vlan{Name: id}, newVlanEntity)
}

func putVlan(ctx context.Context, vlan *ufspb.Vlan, update bool) (*ufspb.Vlan, error) {
	vlan.UpdateTime = ptypes.TimestampNow()
	pm, err := ufsds.Put(ctx, vlan, newVlanEntity, update)
	if err == nil {
		return pm.(*ufspb.Vlan), err
	}
	return nil, err
}

// ImportVlans creates or updates a batch of vlan in datastore
func ImportVlans(ctx context.Context, vlans []*ufspb.Vlan) (*ufsds.OpResults, error) {
	protos := make([]proto.Message, len(vlans))
	utime := ptypes.TimestampNow()
	for i, m := range vlans {
		m.UpdateTime = utime
		protos[i] = m
	}
	return ufsds.Insert(ctx, protos, newVlanEntity, true, true)
}

func queryAllVlan(ctx context.Context) ([]ufsds.FleetEntity, error) {
	var entities []*VlanEntity
	q := datastore.NewQuery(VlanKind)
	if err := datastore.GetAll(ctx, q, &entities); err != nil {
		return nil, err
	}
	fe := make([]ufsds.FleetEntity, len(entities))
	for i, e := range entities {
		fe[i] = e
	}
	return fe, nil
}

// GetAllVlans returns all vlans in datastore.
func GetAllVlans(ctx context.Context) (*ufsds.OpResults, error) {
	return ufsds.GetAll(ctx, queryAllVlan)
}

// QueryVlanByPropertyName query's vlanb Entity in the datastore
func QueryVlanByPropertyName(ctx context.Context, propertyName, id string, keysOnly bool) ([]*ufspb.Vlan, error) {
	q := datastore.NewQuery(VlanKind).KeysOnly(keysOnly).FirestoreMode(true)
	var entities []*VlanEntity
	if err := datastore.GetAll(ctx, q.Eq(propertyName, id), &entities); err != nil {
		logging.Errorf(ctx, "Failed to query from datastore: %s", err)
		return nil, status.Errorf(codes.Internal, ufsds.InternalError)
	}
	if len(entities) == 0 {
		logging.Debugf(ctx, "No vlans found for the query: %s", id)
		return nil, nil
	}
	vlans := make([]*ufspb.Vlan, 0, len(entities))
	for _, entity := range entities {
		if keysOnly {
			vlans = append(vlans, &ufspb.Vlan{
				Name: entity.ID,
			})
		} else {
			pm, perr := entity.GetProto()
			if perr != nil {
				logging.Errorf(ctx, "Failed to unmarshal proto: %s", perr)
				continue
			}
			vlans = append(vlans, pm.(*ufspb.Vlan))
		}
	}
	return vlans, nil
}

// DeleteVlans deletes a batch of vlans
func DeleteVlans(ctx context.Context, resourceNames []string) *ufsds.OpResults {
	protos := make([]proto.Message, len(resourceNames))
	for i, m := range resourceNames {
		protos[i] = &ufspb.Vlan{
			Name: m,
		}
	}
	return ufsds.DeleteAll(ctx, protos, newVlanEntity)
}

// GetVlanIndexedFieldName returns the index name
func GetVlanIndexedFieldName(input string) (string, error) {
	var field string
	input = strings.TrimSpace(input)
	switch strings.ToLower(input) {
	case util.StateFilterName:
		field = "state"
	case util.ZoneFilterName:
		field = "zone"
	case util.TagFilterName:
		field = "tags"
	default:
		return "", status.Errorf(codes.InvalidArgument, "Invalid field name %s - field name for vlan are state/zone/tags", input)
	}
	return field, nil
}
