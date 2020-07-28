// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package configuration

import (
	"context"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ufspb "infra/unifiedfleet/api/v1/proto"
	ufsds "infra/unifiedfleet/app/model/datastore"
)

// ChromePlatformKind is the datastore entity kind for chrome platforms.
const ChromePlatformKind string = "ChromePlatform"

// ChromePlatformEntity is a datastore entity that tracks a platform.
type ChromePlatformEntity struct {
	_kind string `gae:"$kind,ChromePlatform"`
	ID    string `gae:"$id"`
	// ufspb.ChromePlatform cannot be directly used as it contains pointer.
	Platform []byte `gae:",noindex"`
}

// GetProto returns the unmarshaled Chrome platform.
func (e *ChromePlatformEntity) GetProto() (proto.Message, error) {
	var p ufspb.ChromePlatform
	if err := proto.Unmarshal(e.Platform, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func newChromePlatformEntity(ctx context.Context, pm proto.Message) (ufsds.FleetEntity, error) {
	p := pm.(*ufspb.ChromePlatform)
	if p.GetName() == "" {
		return nil, errors.Reason("Empty Chrome Platform ID").Err()
	}
	platform, err := proto.Marshal(p)
	if err != nil {
		return nil, errors.Annotate(err, "fail to marshal ChromePlatform %s", p).Err()
	}
	return &ChromePlatformEntity{
		ID:       p.GetName(),
		Platform: platform,
	}, nil
}

func queryAll(ctx context.Context) ([]ufsds.FleetEntity, error) {
	var entities []*ChromePlatformEntity
	q := datastore.NewQuery(ChromePlatformKind)
	if err := datastore.GetAll(ctx, q, &entities); err != nil {
		return nil, err
	}
	fe := make([]ufsds.FleetEntity, len(entities))
	for i, e := range entities {
		fe[i] = e
	}
	return fe, nil
}

// CreateChromePlatform creates a new chromePlatform in datastore.
func CreateChromePlatform(ctx context.Context, chromePlatform *ufspb.ChromePlatform) (*ufspb.ChromePlatform, error) {
	return putChromePlatform(ctx, chromePlatform, false)
}

// UpdateChromePlatform updates chromePlatform in datastore.
func UpdateChromePlatform(ctx context.Context, chromePlatform *ufspb.ChromePlatform) (*ufspb.ChromePlatform, error) {
	return putChromePlatform(ctx, chromePlatform, true)
}

// GetChromePlatform returns chromePlatform for the given id from datastore.
func GetChromePlatform(ctx context.Context, id string) (*ufspb.ChromePlatform, error) {
	pm, err := ufsds.Get(ctx, &ufspb.ChromePlatform{Name: id}, newChromePlatformEntity)
	if err == nil {
		return pm.(*ufspb.ChromePlatform), err
	}
	return nil, err
}

// ListChromePlatforms lists the chromePlatforms
// Does a query over ChromePlatform entities. Returns up to pageSize entities, plus non-nil cursor (if
// there are more results). pageSize must be positive.
func ListChromePlatforms(ctx context.Context, pageSize int32, pageToken string) (res []*ufspb.ChromePlatform, nextPageToken string, err error) {
	q, err := ufsds.ListQuery(ctx, ChromePlatformKind, pageSize, pageToken, nil, false)
	if err != nil {
		return nil, "", err
	}
	var nextCur datastore.Cursor
	err = datastore.Run(ctx, q, func(ent *ChromePlatformEntity, cb datastore.CursorCB) error {
		pm, err := ent.GetProto()
		if err != nil {
			logging.Errorf(ctx, "Failed to UnMarshal: %s", err)
			return nil
		}
		res = append(res, pm.(*ufspb.ChromePlatform))
		if len(res) >= int(pageSize) {
			if nextCur, err = cb(); err != nil {
				return err
			}
			return datastore.Stop
		}
		return nil
	})
	if err != nil {
		logging.Errorf(ctx, "Failed to List ChromePlatforms %s", err)
		return nil, "", status.Errorf(codes.Internal, ufsds.InternalError)
	}
	if nextCur != nil {
		nextPageToken = nextCur.String()
	}
	return
}

// DeleteChromePlatform deletes the chromePlatform in datastore
func DeleteChromePlatform(ctx context.Context, id string) error {
	return ufsds.Delete(ctx, &ufspb.ChromePlatform{Name: id}, newChromePlatformEntity)
}

// DeleteChromePlatforms deletes a batch of chrome platforms
func DeleteChromePlatforms(ctx context.Context, resourceNames []string) *ufsds.OpResults {
	protos := make([]proto.Message, len(resourceNames))
	for i, m := range resourceNames {
		protos[i] = &ufspb.ChromePlatform{
			Name: m,
		}
	}
	return ufsds.DeleteAll(ctx, protos, newChromePlatformEntity)
}

// ImportChromePlatforms inserts chrome platforms to datastore.
func ImportChromePlatforms(ctx context.Context, platforms []*ufspb.ChromePlatform) (*ufsds.OpResults, error) {
	protos := make([]proto.Message, len(platforms))
	utime := ptypes.TimestampNow()
	for i, p := range platforms {
		p.UpdateTime = utime
		protos[i] = p
	}
	return ufsds.Insert(ctx, protos, newChromePlatformEntity, true, true)
}

// GetAllChromePlatforms returns all platforms in record.
func GetAllChromePlatforms(ctx context.Context) (*ufsds.OpResults, error) {
	return ufsds.GetAll(ctx, queryAll)
}

func putChromePlatform(ctx context.Context, chromePlatform *ufspb.ChromePlatform, update bool) (*ufspb.ChromePlatform, error) {
	chromePlatform.UpdateTime = ptypes.TimestampNow()
	pm, err := ufsds.Put(ctx, chromePlatform, newChromePlatformEntity, update)
	if err == nil {
		return pm.(*ufspb.ChromePlatform), err
	}
	return nil, err
}

// GetChromePlatformIndexedFieldName returns the index name
func GetChromePlatformIndexedFieldName(input string) (string, error) {
	return "", status.Errorf(codes.InvalidArgument, "Invalid field %s - No fields available for ChromePlatform", input)
}
