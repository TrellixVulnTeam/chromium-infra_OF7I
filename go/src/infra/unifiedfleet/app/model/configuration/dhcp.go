// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package configuration

import (
	"context"
	fleet "infra/unifiedfleet/api/v1/proto"
	fleetds "infra/unifiedfleet/app/model/datastore"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DHCPKind is the datastore entity kind dhcp.
const DHCPKind string = "DHCP"

// DHCPEntity is a datastore entity that tracks dhcp.
type DHCPEntity struct {
	_kind string `gae:"$kind,DHCP"`
	// refer to the hostname
	ID   string `gae:"$id"`
	IPv4 string `gae:"ipv4"`
	// fleet.DHCPConfig cannot be directly used as it contains pointer (timestamp).
	Dhcp []byte `gae:",noindex"`
}

// GetProto returns the unmarshaled DHCP.
func (e *DHCPEntity) GetProto() (proto.Message, error) {
	var p fleet.DHCPConfig
	if err := proto.Unmarshal(e.Dhcp, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func newDHCPEntity(ctx context.Context, pm proto.Message) (fleetds.FleetEntity, error) {
	p := pm.(*fleet.DHCPConfig)
	if p.GetHostname() == "" {
		return nil, errors.Reason("Empty hostname in DHCP").Err()
	}
	s, err := proto.Marshal(p)
	if err != nil {
		return nil, errors.Annotate(err, "fail to marshal DHCPConfig %s", p).Err()
	}
	return &DHCPEntity{
		ID:   p.GetHostname(),
		IPv4: p.GetIp(),
		Dhcp: s,
	}, nil
}

// GetDHCPConfig returns dhcp config for the given id from datastore.
func GetDHCPConfig(ctx context.Context, id string) (*fleet.DHCPConfig, error) {
	pm, err := fleetds.Get(ctx, &fleet.DHCPConfig{Hostname: id}, newDHCPEntity)
	if err == nil {
		return pm.(*fleet.DHCPConfig), err
	}
	return nil, err
}

// QueryDHCPConfigByPropertyName query dhcp entity in the datastore.
func QueryDHCPConfigByPropertyName(ctx context.Context, propertyName, id string) ([]*fleet.DHCPConfig, error) {
	q := datastore.NewQuery(DHCPKind)
	var entities []DHCPEntity
	if err := datastore.GetAll(ctx, q.Eq(propertyName, id), &entities); err != nil {
		logging.Errorf(ctx, "Failed to query from datastore: %s", err)
		return nil, status.Errorf(codes.Internal, fleetds.InternalError)
	}
	if len(entities) == 0 {
		logging.Infof(ctx, "No dhcp configs found for the query: %s=%s", propertyName, id)
		return nil, nil
	}
	dhcps := make([]*fleet.DHCPConfig, 0)
	for _, entity := range entities {
		pm, perr := entity.GetProto()
		if perr != nil {
			logging.Errorf(ctx, "Failed to unmarshal proto: %s", perr)
			continue
		}
		dhcps = append(dhcps, pm.(*fleet.DHCPConfig))
	}
	return dhcps, nil
}

// ListDHCPConfigs lists the dhcp configs
//
// Does a query over dhcp config entities. Returns up to pageSize entities, plus non-nil cursor (if
// there are more results). pageSize must be positive.
func ListDHCPConfigs(ctx context.Context, pageSize int32, pageToken string) (res []*fleet.DHCPConfig, nextPageToken string, err error) {
	q, err := fleetds.ListQuery(ctx, DHCPKind, pageSize, pageToken)
	if err != nil {
		return nil, "", err
	}
	var nextCur datastore.Cursor
	err = datastore.Run(ctx, q, func(ent *DHCPEntity, cb datastore.CursorCB) error {
		pm, err := ent.GetProto()
		if err != nil {
			logging.Errorf(ctx, "Failed to UnMarshal: %s", err)
			return nil
		}
		res = append(res, pm.(*fleet.DHCPConfig))
		if len(res) >= int(pageSize) {
			if nextCur, err = cb(); err != nil {
				return err
			}
			return datastore.Stop
		}
		return nil
	})
	if err != nil {
		logging.Errorf(ctx, "Failed to list dhcp configs %s", err)
		return nil, "", status.Errorf(codes.Internal, fleetds.InternalError)
	}
	if nextCur != nil {
		nextPageToken = nextCur.String()
	}
	return
}

// ImportDHCPConfigs creates or updates a batch of dhcp configs in datastore
func ImportDHCPConfigs(ctx context.Context, dhcpConfigs []*fleet.DHCPConfig) (*fleetds.OpResults, error) {
	protos := make([]proto.Message, len(dhcpConfigs))
	utime := ptypes.TimestampNow()
	for i, m := range dhcpConfigs {
		m.UpdateTime = utime
		protos[i] = m
	}
	return fleetds.Insert(ctx, protos, newDHCPEntity, true, true)
}
