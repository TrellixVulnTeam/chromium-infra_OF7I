// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package configuration

import (
	"context"
	"strconv"
	"strings"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/gae/service/datastore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ufspb "infra/unifiedfleet/api/v1/proto"
	ufsds "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/util"
)

// IPKind is the datastore entity kind for IP record
const IPKind string = "IP"

// IPEntity is a datastore entity that tracks IP.
type IPEntity struct {
	_kind string `gae:"$kind,IP"`
	// To avoid duplication, the internal reference ID for IP: vlanName/IPv4, e.g. browser:120/20123455612
	ID       string `gae:"$id"`
	IPv4     uint32 `gae:"ipv4"`
	IPv4Str  string `gae:"ipv4_str"`
	Vlan     string `gae:"vlan"`
	Occupied bool   `gae:"occupied"`
	Reserve  bool   `gae:"reserve"`
}

// GetProto returns the unmarshaled IP.
func (e *IPEntity) GetProto() (proto.Message, error) {
	return &ufspb.IP{
		Id:       e.ID,
		Ipv4:     e.IPv4,
		Ipv4Str:  e.IPv4Str,
		Vlan:     e.Vlan,
		Occupied: e.Occupied,
		Reserve:  e.Reserve,
	}, nil
}

func newIPEntity(ctx context.Context, pm proto.Message) (ufsds.FleetEntity, error) {
	p := pm.(*ufspb.IP)
	if p.GetId() == "" {
		return nil, errors.Reason("Empty hostname in IP").Err()
	}
	if p.GetVlan() == "" {
		return nil, errors.Reason("Empty vlan in IP").Err()
	}
	if p.GetIpv4() == 0 {
		return nil, errors.Reason("Empty ipv4 in IP").Err()
	}
	if p.GetIpv4Str() == "" {
		return nil, errors.Reason("Empty ipv4 str in IP").Err()
	}
	return &IPEntity{
		ID:       p.GetId(),
		IPv4:     p.GetIpv4(),
		IPv4Str:  p.GetIpv4Str(),
		Vlan:     p.GetVlan(),
		Occupied: p.GetOccupied(),
		Reserve:  p.GetReserve(),
	}, nil
}

func newDeleteIPEntity(ctx context.Context, pm proto.Message) (ufsds.FleetEntity, error) {
	p := pm.(*ufspb.IP)
	if p.GetId() == "" {
		return nil, errors.Reason("Empty IP id").Err()
	}
	return &IPEntity{
		ID:       p.GetId(),
		IPv4:     p.GetIpv4(),
		IPv4Str:  p.GetIpv4Str(),
		Vlan:     p.GetVlan(),
		Occupied: p.GetOccupied(),
		Reserve:  p.GetReserve(),
	}, nil
}

// QueryIPByPropertyName query IP Entity by property in the datastore
func QueryIPByPropertyName(ctx context.Context, propertyMap map[string]string) ([]*ufspb.IP, error) {
	q := datastore.NewQuery(IPKind).FirestoreMode(true)
	var entities []*IPEntity
	for propertyName, id := range propertyMap {
		switch propertyName {
		case "ipv4":
			u64, err := strconv.ParseUint(id, 10, 32)
			if err != nil {
				logging.Errorf(ctx, "Failed to convert the property 'ipv4' %s to uint64", id)
				return nil, status.Errorf(codes.InvalidArgument, "%s for %q: %s", ufsds.InvalidArgument, propertyName, err.Error())
			}
			q = q.Eq(propertyName, uint32(u64))
		case "occupied":
			b, err := strconv.ParseBool(id)
			if err != nil {
				logging.Errorf(ctx, "Failed to convert the property 'occupied' %s to bool", id)
				return nil, status.Errorf(codes.InvalidArgument, "%s for %q: %s", ufsds.InvalidArgument, propertyName, err.Error())
			}
			q = q.Eq(propertyName, b)
		case "reserve":
			b, err := strconv.ParseBool(id)
			if err != nil {
				logging.Errorf(ctx, "Failed to convert the property 'reserve' %s to bool", id)
				return nil, status.Errorf(codes.InvalidArgument, "%s for %q: %s", ufsds.InvalidArgument, propertyName, err.Error())
			}
			q = q.Eq(propertyName, b)
		default:
			q = q.Eq(propertyName, id)
		}
	}
	if err := datastore.GetAll(ctx, q, &entities); err != nil {
		logging.Errorf(ctx, "Failed to query from datastore: %s", err)
		return nil, status.Errorf(codes.Internal, ufsds.InternalError)
	}
	if len(entities) == 0 {
		logging.Infof(ctx, "No ips found for the query: %#v", propertyMap)
		return nil, nil
	}
	ips := make([]*ufspb.IP, 0, len(entities))
	for _, entity := range entities {
		pm, perr := entity.GetProto()
		if perr != nil {
			logging.Errorf(ctx, "Failed to unmarshal proto: %s", perr)
			continue
		}
		ips = append(ips, pm.(*ufspb.IP))
	}
	return ips, nil
}

// ListIPs lists the ips
//
// Does a query over ip entities. Returns up to pageSize entities, plus non-nil cursor (if
// there are more results). pageSize must be positive.
func ListIPs(ctx context.Context, pageSize int32, pageToken string, filterMap map[string][]interface{}, keysOnly bool) (res []*ufspb.IP, nextPageToken string, err error) {
	q, err := ufsds.ListQuery(ctx, IPKind, pageSize, pageToken, filterMap, keysOnly)
	if err != nil {
		return nil, "", err
	}
	var nextCur datastore.Cursor
	err = datastore.Run(ctx, q, func(ent *IPEntity, cb datastore.CursorCB) error {
		if keysOnly {
			ip := &ufspb.IP{
				Id: ent.ID,
			}
			res = append(res, ip)
		} else {
			pm, err := ent.GetProto()
			if err != nil {
				logging.Errorf(ctx, "Failed to UnMarshal: %s", err)
				return nil
			}
			res = append(res, pm.(*ufspb.IP))
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
		logging.Errorf(ctx, "Failed to list ips %s", err)
		return nil, "", status.Errorf(codes.Internal, ufsds.InternalError)
	}
	if nextCur != nil {
		nextPageToken = nextCur.String()
	}
	return
}

func queryAllIP(ctx context.Context) ([]ufsds.FleetEntity, error) {
	var entities []*IPEntity
	q := datastore.NewQuery(IPKind)
	if err := datastore.GetAll(ctx, q, &entities); err != nil {
		return nil, err
	}
	fe := make([]ufsds.FleetEntity, len(entities))
	for i, e := range entities {
		fe[i] = e
	}
	return fe, nil
}

// GetAllIPs returns all ips in datastore.
func GetAllIPs(ctx context.Context) (*ufsds.OpResults, error) {
	return ufsds.GetAll(ctx, queryAllIP)
}

// ImportIPs creates or updates a batch of ips in datastore
func ImportIPs(ctx context.Context, ips []*ufspb.IP) (*ufsds.OpResults, error) {
	protos := make([]proto.Message, len(ips))
	for i, m := range ips {
		protos[i] = m
	}
	return ufsds.Insert(ctx, protos, newIPEntity, true, true)
}

// DeleteIPs deletes a batch of ips
func DeleteIPs(ctx context.Context, resourceNames []string) *ufsds.OpResults {
	protos := make([]proto.Message, len(resourceNames))
	for i, m := range resourceNames {
		protos[i] = &ufspb.IP{
			Id: m,
		}
	}
	return ufsds.DeleteAll(ctx, protos, newDeleteIPEntity)
}

// BatchDeleteIPs deletes ips in datastore.
//
// This is a non-atomic operation. Must be used within a transaction.
// Will lead to partial deletes if not used in a transaction.
func BatchDeleteIPs(ctx context.Context, ids []string) error {
	checkEntities := make([]*IPEntity, 0, len(ids))
	for _, id := range ids {
		entity := &IPEntity{
			ID: id,
		}
		checkEntities = append(checkEntities, entity)
	}
	if err := datastore.Delete(ctx, checkEntities); err != nil {
		return status.Errorf(codes.Internal, err.Error())
	}
	return nil
}

// GetIPIndexedFieldName returns the index name
func GetIPIndexedFieldName(input string) (string, error) {
	var field string
	input = strings.TrimSpace(input)
	switch strings.ToLower(input) {
	case util.IPV4FilterName:
		field = "ipv4"
	case util.IPV4StringFilterName:
		field = "ipv4_str"
	case util.VlanFilterName:
		field = "vlan"
	case util.OccupiedFilterName:
		field = "occupied"
	default:
		return "", status.Errorf(codes.InvalidArgument, "Invalid field name %s - field name for IP are ipv4/ipv4str/vlan/occupied", input)
	}
	return field, nil
}

// BatchUpdateIPs updates the ip entity to UFS.
//
// This can be used inside a transaction
func BatchUpdateIPs(ctx context.Context, ips []*ufspb.IP) ([]*ufspb.IP, error) {
	protos := make([]proto.Message, len(ips))
	for i, ip := range ips {
		protos[i] = ip
	}
	_, err := ufsds.PutAll(ctx, protos, newIPEntity, true)
	if err == nil {
		return ips, err
	}
	return nil, err
}
