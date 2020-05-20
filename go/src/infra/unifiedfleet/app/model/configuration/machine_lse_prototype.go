// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package configuration

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fleet "infra/unifiedfleet/api/v1/proto"
	fleetds "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/inventory"
)

// MachineLSEPrototypeKind is the datastore entity kind for chrome platforms.
const MachineLSEPrototypeKind string = "MachineLSEPrototype"

// MachineLSEPrototypeEntity is a datastore entity that tracks a platform.
type MachineLSEPrototypeEntity struct {
	_kind string `gae:"$kind,MachineLSEPrototype"`
	ID    string `gae:"$id"`
	// fleet.MachineLSEPrototype cannot be directly used as it contains pointer.
	MachineLSEPrototype []byte `gae:",noindex"`
}

// GetProto returns the unmarshaled Chrome platform.
func (e *MachineLSEPrototypeEntity) GetProto() (proto.Message, error) {
	var p fleet.MachineLSEPrototype
	if err := proto.Unmarshal(e.MachineLSEPrototype, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func newMachineLSEPrototypeEntity(ctx context.Context, pm proto.Message) (fleetds.FleetEntity, error) {
	p := pm.(*fleet.MachineLSEPrototype)
	if p.GetName() == "" {
		return nil, errors.Reason("Empty Chrome Platform ID").Err()
	}
	machineLSEPrototype, err := proto.Marshal(p)
	if err != nil {
		return nil, errors.Annotate(err, "fail to marshal MachineLSEPrototype %s", p).Err()
	}
	return &MachineLSEPrototypeEntity{
		ID:                  p.GetName(),
		MachineLSEPrototype: machineLSEPrototype,
	}, nil
}

// CreateMachineLSEPrototype creates a new machineLSEPrototype in datastore.
func CreateMachineLSEPrototype(ctx context.Context, machineLSEPrototype *fleet.MachineLSEPrototype) (*fleet.MachineLSEPrototype, error) {
	return putMachineLSEPrototype(ctx, machineLSEPrototype, false)
}

// UpdateMachineLSEPrototype updates machineLSEPrototype in datastore.
func UpdateMachineLSEPrototype(ctx context.Context, machineLSEPrototype *fleet.MachineLSEPrototype) (*fleet.MachineLSEPrototype, error) {
	return putMachineLSEPrototype(ctx, machineLSEPrototype, true)
}

// GetMachineLSEPrototype returns machineLSEPrototype for the given id from datastore.
func GetMachineLSEPrototype(ctx context.Context, id string) (*fleet.MachineLSEPrototype, error) {
	pm, err := fleetds.Get(ctx, &fleet.MachineLSEPrototype{Name: id}, newMachineLSEPrototypeEntity)
	if err == nil {
		return pm.(*fleet.MachineLSEPrototype), err
	}
	return nil, err
}

// ListMachineLSEPrototypes lists the machineLSEPrototypes
//
// Does a query over MachineLSEPrototype entities. Returns up to pageSize entities, plus non-nil cursor (if
// there are more results). pageSize must be positive.
func ListMachineLSEPrototypes(ctx context.Context, pageSize int32, pageToken string) (res []*fleet.MachineLSEPrototype, nextPageToken string, err error) {
	q, err := fleetds.ListQuery(ctx, MachineLSEPrototypeKind, pageSize, pageToken)
	if err != nil {
		return nil, "", err
	}
	var nextCur datastore.Cursor
	err = datastore.Run(ctx, q, func(ent *MachineLSEPrototypeEntity, cb datastore.CursorCB) error {
		pm, err := ent.GetProto()
		if err != nil {
			logging.Errorf(ctx, "Failed to UnMarshal: %s", err)
			return nil
		}
		res = append(res, pm.(*fleet.MachineLSEPrototype))
		if len(res) >= int(pageSize) {
			if nextCur, err = cb(); err != nil {
				return err
			}
			return datastore.Stop
		}
		return nil
	})
	if err != nil {
		logging.Errorf(ctx, "Failed to List MachineLSEPrototypes %s", err)
		return nil, "", status.Errorf(codes.Internal, fleetds.InternalError)
	}
	if nextCur != nil {
		nextPageToken = nextCur.String()
	}
	return
}

// DeleteMachineLSEPrototype deletes the machineLSEPrototype in datastore
//
// For referential data intergrity,
// Delete if there are no references to the MachineLSEPrototype by other resources in the datastore.
// If there are any references, delete will be rejected and an error message will be thrown.
func DeleteMachineLSEPrototype(ctx context.Context, id string) error {
	machinelses, err := inventory.QueryMachineLSEByPropertyName(ctx, "machinelse_prototype_id", id, true)
	if err != nil {
		return err
	}
	if len(machinelses) > 0 {
		var errorMsg strings.Builder
		errorMsg.WriteString(fmt.Sprintf("MachineLSEPrototype %s cannot be deleted because there are other resources which are referring this MachineLSEPrototype.", id))
		errorMsg.WriteString(fmt.Sprintf("\nMachineLSEs referring the MachineLSEPrototype:\n"))
		for _, machinelse := range machinelses {
			errorMsg.WriteString(machinelse.Name + ", ")
		}
		logging.Infof(ctx, errorMsg.String())
		return status.Errorf(codes.FailedPrecondition, errorMsg.String())
	}
	return fleetds.Delete(ctx, &fleet.MachineLSEPrototype{Name: id}, newMachineLSEPrototypeEntity)
}

func putMachineLSEPrototype(ctx context.Context, machineLSEPrototype *fleet.MachineLSEPrototype, update bool) (*fleet.MachineLSEPrototype, error) {
	machineLSEPrototype.UpdateTime = ptypes.TimestampNow()
	pm, err := fleetds.Put(ctx, machineLSEPrototype, newMachineLSEPrototypeEntity, update)
	if err == nil {
		return pm.(*fleet.MachineLSEPrototype), err
	}
	return nil, err
}

// ImportMachineLSEPrototypes creates or updates a batch of machine lse prototypes in datastore
func ImportMachineLSEPrototypes(ctx context.Context, lps []*fleet.MachineLSEPrototype) (*fleetds.OpResults, error) {
	protos := make([]proto.Message, len(lps))
	utime := ptypes.TimestampNow()
	for i, m := range lps {
		m.UpdateTime = utime
		protos[i] = m
	}
	return fleetds.Insert(ctx, protos, newMachineLSEPrototypeEntity, true, true)
}
