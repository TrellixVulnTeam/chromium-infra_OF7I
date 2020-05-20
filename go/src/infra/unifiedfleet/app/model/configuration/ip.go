// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package configuration

import (
	"context"
	fleet "infra/unifiedfleet/api/v1/proto"
	fleetds "infra/unifiedfleet/app/model/datastore"
	"strconv"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// IPKind is the datastore entity kind for IP record
const IPKind string = "IP"

// IPEntity is a datastore entity that tracks IP.
type IPEntity struct {
	_kind string `gae:"$kind,IP"`
	// The internal reference ID for IP: vlan/IPv4
	ID       string `gae:"$id"`
	IPv4     uint32 `gae:"ipv4"`
	Vlan     string `gae:"vlan"`
	Occupied bool   `gae:"occupied"`
}

// GetProto returns the unmarshaled IP.
func (e *IPEntity) GetProto() (proto.Message, error) {
	return &fleet.IP{
		Id:       e.ID,
		Ipv4:     e.IPv4,
		Vlan:     e.Vlan,
		Occupied: e.Occupied,
	}, nil
}

func newIPEntity(ctx context.Context, pm proto.Message) (fleetds.FleetEntity, error) {
	p := pm.(*fleet.IP)
	if p.GetId() == "" {
		return nil, errors.Reason("Empty hostname in IP").Err()
	}
	if p.GetVlan() == "" {
		return nil, errors.Reason("Empty vlan in IP").Err()
	}
	if p.GetIpv4() == 0 {
		return nil, errors.Reason("Empty ipv4 in IP").Err()
	}
	return &IPEntity{
		ID:       p.GetId(),
		IPv4:     p.GetIpv4(),
		Vlan:     p.GetVlan(),
		Occupied: p.GetOccupied(),
	}, nil
}

// QueryIPByPropertyName query IP Entity by property in the datastore
func QueryIPByPropertyName(ctx context.Context, propertyName, id string) ([]*fleet.IP, error) {
	q := datastore.NewQuery(IPKind).FirestoreMode(true)
	var entities []*IPEntity
	switch propertyName {
	case "ipv4":
		u64, err := strconv.ParseUint(id, 10, 32)
		if err != nil {
			logging.Errorf(ctx, "Failed to convert the property 'ipv4' %s to uint64", id)
			return nil, status.Errorf(codes.InvalidArgument, "%s for %q: %s", fleetds.InvalidArgument, propertyName, err.Error())
		}
		q = q.Eq(propertyName, uint32(u64))
	case "occupied":
		b, err := strconv.ParseBool(id)
		if err != nil {
			logging.Errorf(ctx, "Failed to convert the property 'occupied' %s to bool", id)
			return nil, status.Errorf(codes.InvalidArgument, "%s for %q: %s", fleetds.InvalidArgument, propertyName, err.Error())
		}
		q = q.Eq(propertyName, b)
	default:
		q = q.Eq(propertyName, id)
	}
	if err := datastore.GetAll(ctx, q, &entities); err != nil {
		logging.Errorf(ctx, "Failed to query from datastore: %s", err)
		return nil, status.Errorf(codes.Internal, fleetds.InternalError)
	}
	if len(entities) == 0 {
		logging.Infof(ctx, "No ips found for the query: %s", id)
		return nil, nil
	}
	ips := make([]*fleet.IP, 0, len(entities))
	for _, entity := range entities {
		pm, perr := entity.GetProto()
		if perr != nil {
			logging.Errorf(ctx, "Failed to unmarshal proto: %s", perr)
			continue
		}
		ips = append(ips, pm.(*fleet.IP))
	}
	return ips, nil
}

// ImportIPs creates or updates a batch of ips in datastore
func ImportIPs(ctx context.Context, ips []*fleet.IP) (*fleetds.OpResults, error) {
	protos := make([]proto.Message, len(ips))
	for i, m := range ips {
		protos[i] = m
	}
	return fleetds.Insert(ctx, protos, newIPEntity, true, true)
}
