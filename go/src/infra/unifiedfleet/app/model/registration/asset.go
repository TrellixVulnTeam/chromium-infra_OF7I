// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package registration

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

// AssetKind is a datastore entity identifier for Asset
const AssetKind string = "Asset"

// AssetEntity is a datastore entity that tracks Assets
type AssetEntity struct {
	_kind       string   `gae:"$kind,Asset"`
	Name        string   `gae:"$id"`
	Zone        string   `gae:"zone"`
	Type        string   `gae:"type"`
	Model       string   `gae:"model"`
	Rack        string   `gae:"rack"`
	BuildTarget string   `gae:"build_target"`
	Phase       string   `gae:"phase"`
	Tags        []string `gae:"tags"`
	Asset       []byte   `gae:",noindex"` // Marshalled Asset proto
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
	a, ok := pm.(*ufspb.Asset)
	if !ok {
		return nil, errors.Reason("Invalid asset (proto message is probably not asset or nil)").Err()
	}
	if a.GetName() == "" {
		return nil, errors.Reason("Empty Asset ID").Err()
	}
	asset, err := proto.Marshal(a)
	if err != nil {
		return nil, errors.Annotate(err, "Failed to marshal asset %s", a).Err()
	}
	return &AssetEntity{
		Name:        a.GetName(),
		Zone:        a.GetLocation().GetZone().String(),
		Type:        a.GetType().String(),
		Model:       a.GetModel(),
		Rack:        a.GetLocation().GetRack(),
		BuildTarget: a.GetInfo().GetBuildTarget(),
		Phase:       a.GetInfo().GetPhase(),
		Tags:        a.GetTags(),
		Asset:       asset,
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

// GetAllAssets returns all assets currently in the datastore.
func GetAllAssets(ctx context.Context) ([]*ufspb.Asset, error) {
	resp, err := ufsds.GetAll(ctx, func(ctx context.Context) ([]ufsds.FleetEntity, error) {
		var entities []*AssetEntity
		q := datastore.NewQuery(AssetKind)
		if err := datastore.GetAll(ctx, q, &entities); err != nil {
			return nil, err
		}
		fe := make([]ufsds.FleetEntity, len(entities))
		for i, e := range entities {
			fe[i] = e
		}
		return fe, nil
	})
	if err != nil {
		return nil, err
	}
	assets := make([]*ufspb.Asset, 0, len(*resp))
	for _, a := range *resp {
		if b, ok := a.Data.(*ufspb.Asset); ok && a.Err == nil {
			assets = append(assets, b)
		}
	}
	return assets, nil
}

// DeleteAsset deletes the asset corresponding to id from datastore.
func DeleteAsset(ctx context.Context, id string) error {
	return ufsds.Delete(ctx, &ufspb.Asset{Name: id}, newAssetEntity)
}

// CreateAsset creates an asset record in the datastore using the given asset proto.
func CreateAsset(ctx context.Context, asset *ufspb.Asset) (*ufspb.Asset, error) {
	if asset == nil || asset.Name == "" || asset.Type == ufspb.AssetType_UNDEFINED || asset.Location == nil {
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

// BatchUpdateAssets updates the assets to the datastore
func BatchUpdateAssets(ctx context.Context, assets []*ufspb.Asset) ([]*ufspb.Asset, error) {
	protos := make([]proto.Message, len(assets))
	updateTime := ptypes.TimestampNow()
	for i, asset := range assets {
		if asset != nil {
			asset.UpdateTime = updateTime
			protos[i] = asset
		}
	}
	_, err := ufsds.PutAll(ctx, protos, newAssetEntity, false)
	if err == nil {
		return assets, nil
	}
	return nil, err
}

// GetAssetIndexedFieldName returns the index name
func GetAssetIndexedFieldName(input string) (string, error) {
	var field string
	input = strings.TrimSpace(input)
	switch strings.ToLower(input) {
	case util.ZoneFilterName:
		field = "zone"
	case util.RackFilterName:
		field = "rack"
	case util.ModelFilterName:
		field = "model"
	case util.BoardFilterName:
		field = "build_target"
	case util.AssetTypeFilterName:
		field = "type"
	case util.PhaseFilterName:
		field = "phase"
	case util.TagFilterName:
		field = "tags"
	default:
		return "", status.Errorf(codes.InvalidArgument, "Invalid field name %s - field name for asset are zone/rack/model/buildtarget(board)/assettype/phase/tags", input)
	}
	return field, nil
}
