// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package configuration

import (
	"context"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ufspb "infra/unifiedfleet/api/v1/proto"
	ufsds "infra/unifiedfleet/app/model/datastore"
)

// OSVersionKind is the datastore entity kind for chrome os_version.
const OSVersionKind string = "OSVersion"

// OSVersionEntity is a datastore entity that tracks an os_version.
type OSVersionEntity struct {
	_kind string `gae:"$kind,OSVersion"`
	ID    string `gae:"$id"`
	// fleet.OSVersion cannot be directly used as it contains pointer.
	OSVersion []byte `gae:",noindex"`
}

// GetProto returns the unmarshaled Chrome os_version.
func (e *OSVersionEntity) GetProto() (proto.Message, error) {
	var p ufspb.OSVersion
	if err := proto.Unmarshal(e.OSVersion, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func newOSVersionEntity(ctx context.Context, pm proto.Message) (ufsds.FleetEntity, error) {
	p := pm.(*ufspb.OSVersion)
	if p.GetValue() == "" {
		return nil, errors.Reason("Empty name for os version").Err()
	}
	msg, err := proto.Marshal(p)
	if err != nil {
		return nil, errors.Annotate(err, "fail to marshal os version %s", p).Err()
	}
	return &OSVersionEntity{
		ID:        p.GetValue(),
		OSVersion: msg,
	}, nil
}

func queryAllOS(ctx context.Context) ([]ufsds.FleetEntity, error) {
	var entities []*OSVersionEntity
	q := datastore.NewQuery(OSVersionKind)
	if err := datastore.GetAll(ctx, q, &entities); err != nil {
		return nil, err
	}
	fe := make([]ufsds.FleetEntity, len(entities))
	for i, e := range entities {
		fe[i] = e
	}
	return fe, nil
}

// ListOSes lists the chrome os_versions
func ListOSes(ctx context.Context, pageSize int32, pageToken string) (res []*ufspb.OSVersion, nextPageToken string, err error) {
	q, err := ufsds.ListQuery(ctx, OSVersionKind, pageSize, pageToken, nil, false)
	if err != nil {
		return nil, "", err
	}
	var nextCur datastore.Cursor
	err = datastore.Run(ctx, q, func(ent *OSVersionEntity, cb datastore.CursorCB) error {
		pm, err := ent.GetProto()
		if err != nil {
			logging.Errorf(ctx, "Failed to UnMarshal: %s", err)
			return nil
		}
		res = append(res, pm.(*ufspb.OSVersion))
		if len(res) >= int(pageSize) {
			if nextCur, err = cb(); err != nil {
				return err
			}
			return datastore.Stop
		}
		return nil
	})
	if err != nil {
		logging.Errorf(ctx, "Failed to List os versions %s", err)
		return nil, "", status.Errorf(codes.Internal, ufsds.InternalError)
	}
	if nextCur != nil {
		nextPageToken = nextCur.String()
	}
	return
}

// ImportOses inserts chrome os versions to datastore.
func ImportOses(ctx context.Context, oses []*ufspb.OSVersion) (*ufsds.OpResults, error) {
	protos := make([]proto.Message, len(oses))
	for i, p := range oses {
		protos[i] = p
	}
	return ufsds.Insert(ctx, protos, newOSVersionEntity, true, true)
}

// GetAllOSes returns all os versions in record.
func GetAllOSes(ctx context.Context) (*ufsds.OpResults, error) {
	return ufsds.GetAll(ctx, queryAllOS)
}

// DeleteOSes deletes a batch of chrome os_version
func DeleteOSes(ctx context.Context, resourceNames []string) *ufsds.OpResults {
	protos := make([]proto.Message, len(resourceNames))
	for i, m := range resourceNames {
		protos[i] = &ufspb.OSVersion{
			Value: m,
		}
	}
	return ufsds.DeleteAll(ctx, protos, newOSVersionEntity)
}

// GetOSVersionIndexedFieldName returns the index name
func GetOSVersionIndexedFieldName(input string) (string, error) {
	return "", status.Errorf(codes.InvalidArgument, "Invalid field %s - No fields available for OSVersion", input)
}
