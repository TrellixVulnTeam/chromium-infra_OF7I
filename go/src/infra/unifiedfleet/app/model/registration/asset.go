// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package registration

import (
	"context"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/gae/service/datastore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ufspb "infra/unifiedfleet/api/v1/proto"
	ufsds "infra/unifiedfleet/app/model/datastore"
)

// AssetKind is a datastore entity identifier for Asset
const AssetKind string = "Asset"

// AssetEntity is a datastore entity that tracks Assets
type AssetEntity struct {
	_kind string `gae:"$kind,Asset"`
	Name  string `gae:"$id"`
	Zone  string `gae:"zone"`
	Type  string `gae:"type"`
	Model string `gae:"model"`
	Asset []byte `gae:",noindex"` // Marshalled Asset proto
}

// GetProto returns unmarshalled Asset.
func (a *AssetEntity) GetProto() (proto.Message, error) {
	var p ufspb.Asset
	if err := proto.Unmarshal(a.Asset, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// newAssetEntity creates a new asset entity object from proto message.
func newAssetEntity(ctx context.Context, pm proto.Message) (ufsds.FleetEntity, error) {
	a := pm.(*ufspb.Asset)
	if a.GetName() == "" {
		return nil, errors.Reason("Empty Asset ID").Err()
	}
	asset, err := proto.Marshal(a)
	if err != nil {
		return nil, errors.Annotate(err, "Failed to marshal asset %s", a).Err()
	}
	return &AssetEntity{
		Name:  a.GetName(),
		Zone:  a.GetLocation().GetZone().String(),
		Type:  a.GetType(),
		Model: a.GetModel(),
		Asset: asset,
	}, nil
}

// GetAsset returns asset corresponding to the name.
func GetAsset(ctx context.Context, name string) (*ufspb.Asset, error) {
	pm, err := ufsds.Get(ctx, &ufspb.Asset{Name: name}, newAssetEntity)
	if err != nil {
		return nil, err
	}
	return pm.(*ufspb.Asset), err
}

// DeleteAsset deletes the asset corresponding to id from datastore.
func DeleteAsset(ctx context.Context, id string) error {
	return ufsds.Delete(ctx, &ufspb.Asset{Name: id}, newAssetEntity)
}

// CreateAsset creates an asset record in the datastore using the given asset proto.
func CreateAsset(ctx context.Context, asset *ufspb.Asset) (*ufspb.Asset, error) {
	if asset == nil || asset.Name == "" || asset.Type == "" || asset.Model == "" || asset.Location == nil {
		return nil, errors.Reason("Invalid Asset [Asset is empty or one or more required fields are missing]").Err()
	}
	asset.UpdateTime = ptypes.TimestampNow()
	pm, err := ufsds.Put(ctx, asset, newAssetEntity, false)
	if err != nil {
		return nil, err
	}
	return pm.(*ufspb.Asset), nil
}

// UpdateAsset updates the asset to the given asset proto.
func UpdateAsset(ctx context.Context, asset *ufspb.Asset) (*ufspb.Asset, error) {
	asset.UpdateTime = ptypes.TimestampNow()
	pm, err := ufsds.Put(ctx, asset, newAssetEntity, true)
	if err != nil {
		return nil, err
	}
	return pm.(*ufspb.Asset), nil
}

// ListAssets lists the assets
// Does a query over asset entities. Returns pageSize number of entities and a
// non-nil cursor if there are more results. pageSize must be positive
func ListAssets(ctx context.Context, pageSize int32, pageToken string, filterMap map[string][]interface{}, keysOnly bool) (res []*ufspb.Asset, nextPageToken string, err error) {
	q, err := ufsds.ListQuery(ctx, AssetKind, pageSize, pageToken, filterMap, keysOnly)
	if err != nil {
		return nil, "", err
	}

	var nextCur datastore.Cursor
	err = datastore.Run(ctx, q, func(ent *AssetEntity, cb datastore.CursorCB) error {
		if keysOnly {
			asset := &ufspb.Asset{
				Name: ent.Name,
			}
			res = append(res, asset)
		} else {
			pm, err := ent.GetProto()
			if err != nil {
				logging.Errorf(ctx, "Failed to unmarshall asset: %s", err)
				return nil
			}
			res = append(res, pm.(*ufspb.Asset))
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
		logging.Errorf(ctx, "Failed to list assets %s", err)
		return nil, "", status.Errorf(codes.Internal, ufsds.InternalError)
	}
	if nextCur != nil {
		nextPageToken = nextCur.String()
	}
	return
}
