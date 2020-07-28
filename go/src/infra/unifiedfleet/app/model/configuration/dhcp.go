// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package configuration

import (
	"context"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ufspb "infra/unifiedfleet/api/v1/proto"
	ufsds "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/util"
)

// DHCPKind is the datastore entity kind dhcp.
const DHCPKind string = "DHCP"

// DHCPEntity is a datastore entity that tracks dhcp.
type DHCPEntity struct {
	_kind string `gae:"$kind,DHCP"`
	// refer to the hostname
	ID   string `gae:"$id"`
	IPv4 string `gae:"ipv4"`
	// ufspb.DHCPConfig cannot be directly used as it contains pointer (timestamp).
	Dhcp []byte `gae:",noindex"`
}

// GetProto returns the unmarshaled DHCP.
func (e *DHCPEntity) GetProto() (proto.Message, error) {
	var p ufspb.DHCPConfig
	if err := proto.Unmarshal(e.Dhcp, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func newDHCPEntity(ctx context.Context, pm proto.Message) (ufsds.FleetEntity, error) {
	p := pm.(*ufspb.DHCPConfig)
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
func GetDHCPConfig(ctx context.Context, id string) (*ufspb.DHCPConfig, error) {
	pm, err := ufsds.Get(ctx, &ufspb.DHCPConfig{Hostname: id}, newDHCPEntity)
	if err == nil {
		return pm.(*ufspb.DHCPConfig), err
	}
	return nil, err
}

// QueryDHCPConfigByPropertyName query dhcp entity in the datastore.
func QueryDHCPConfigByPropertyName(ctx context.Context, propertyName, id string) ([]*ufspb.DHCPConfig, error) {
	q := datastore.NewQuery(DHCPKind)
	var entities []DHCPEntity
	if err := datastore.GetAll(ctx, q.Eq(propertyName, id), &entities); err != nil {
		logging.Errorf(ctx, "Failed to query from datastore: %s", err)
		return nil, status.Errorf(codes.Internal, ufsds.InternalError)
	}
	if len(entities) == 0 {
		logging.Infof(ctx, "No dhcp configs found for the query: %s=%s", propertyName, id)
		return nil, nil
	}
	dhcps := make([]*ufspb.DHCPConfig, 0)
	for _, entity := range entities {
		pm, perr := entity.GetProto()
		if perr != nil {
			logging.Errorf(ctx, "Failed to unmarshal proto: %s", perr)
			continue
		}
		dhcps = append(dhcps, pm.(*ufspb.DHCPConfig))
	}
	return dhcps, nil
}

// ListDHCPConfigs lists the dhcp configs
//
// Does a query over dhcp config entities. Returns up to pageSize entities, plus non-nil cursor (if
// there are more results). pageSize must be positive.
func ListDHCPConfigs(ctx context.Context, pageSize int32, pageToken string, filterMap map[string][]interface{}, keysOnly bool) (res []*ufspb.DHCPConfig, nextPageToken string, err error) {
	q, err := ufsds.ListQuery(ctx, DHCPKind, pageSize, pageToken, filterMap, keysOnly)
	if err != nil {
		return nil, "", err
	}
	var nextCur datastore.Cursor
	err = datastore.Run(ctx, q, func(ent *DHCPEntity, cb datastore.CursorCB) error {
		if keysOnly {
			dhcpConfig := &ufspb.DHCPConfig{
				Hostname: ent.ID,
			}
			res = append(res, dhcpConfig)
		} else {
			pm, err := ent.GetProto()
			if err != nil {
				logging.Errorf(ctx, "Failed to UnMarshal: %s", err)
				return nil
			}
			res = append(res, pm.(*ufspb.DHCPConfig))
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
		logging.Errorf(ctx, "Failed to list dhcp configs %s", err)
		return nil, "", status.Errorf(codes.Internal, ufsds.InternalError)
	}
	if nextCur != nil {
		nextPageToken = nextCur.String()
	}
	return
}

// ImportDHCPConfigs creates or updates a batch of dhcp configs in datastore
func ImportDHCPConfigs(ctx context.Context, dhcpConfigs []*ufspb.DHCPConfig) (*ufsds.OpResults, error) {
	protos := make([]proto.Message, len(dhcpConfigs))
	utime := ptypes.TimestampNow()
	for i, m := range dhcpConfigs {
		m.UpdateTime = utime
		protos[i] = m
	}
	return ufsds.Insert(ctx, protos, newDHCPEntity, true, true)
}

func queryAllDHCP(ctx context.Context) ([]ufsds.FleetEntity, error) {
	var entities []*DHCPEntity
	q := datastore.NewQuery(DHCPKind)
	if err := datastore.GetAll(ctx, q, &entities); err != nil {
		return nil, err
	}
	fe := make([]ufsds.FleetEntity, len(entities))
	for i, e := range entities {
		fe[i] = e
	}
	return fe, nil
}

// GetAllDHCPs returns all dhcps in datastore.
func GetAllDHCPs(ctx context.Context) (*ufsds.OpResults, error) {
	return ufsds.GetAll(ctx, queryAllDHCP)
}

// DeleteDHCPs deletes a batch of dhcps
func DeleteDHCPs(ctx context.Context, resourceNames []string) *ufsds.OpResults {
	protos := make([]proto.Message, len(resourceNames))
	for i, m := range resourceNames {
		protos[i] = &ufspb.DHCPConfig{
			Hostname: m,
		}
	}
	return ufsds.DeleteAll(ctx, protos, newDHCPEntity)
}

// GetDHCPIndexedFieldName returns the index name
func GetDHCPIndexedFieldName(input string) (string, error) {
	var field string
	input = strings.TrimSpace(input)
	switch strings.ToLower(input) {
	case util.IPV4FilterName:
		field = "ipv4"
	default:
		return "", status.Errorf(codes.InvalidArgument, "Invalid field name %s - field name for DHCP are ipv4", input)
	}
	return field, nil
}
