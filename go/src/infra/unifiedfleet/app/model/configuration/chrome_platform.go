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
	"infra/unifiedfleet/app/model/registration"
)

// ChromePlatformKind is the datastore entity kind for chrome platforms.
const ChromePlatformKind string = "ChromePlatform"

// ChromePlatformEntity is a datastore entity that tracks a platform.
type ChromePlatformEntity struct {
	_kind string `gae:"$kind,ChromePlatform"`
	ID    string `gae:"$id"`
	// fleet.ChromePlatform cannot be directly used as it contains pointer.
	Platform []byte `gae:",noindex"`
}

// GetProto returns the unmarshaled Chrome platform.
func (e *ChromePlatformEntity) GetProto() (proto.Message, error) {
	var p fleet.ChromePlatform
	if err := proto.Unmarshal(e.Platform, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func newChromePlatformEntity(ctx context.Context, pm proto.Message) (fleetds.FleetEntity, error) {
	p := pm.(*fleet.ChromePlatform)
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

func queryAll(ctx context.Context) ([]fleetds.FleetEntity, error) {
	var entities []*ChromePlatformEntity
	q := datastore.NewQuery(ChromePlatformKind)
	if err := datastore.GetAll(ctx, q, &entities); err != nil {
		return nil, err
	}
	fe := make([]fleetds.FleetEntity, len(entities))
	for i, e := range entities {
		fe[i] = e
	}
	return fe, nil
}

// CreateChromePlatform creates a new chromePlatform in datastore.
func CreateChromePlatform(ctx context.Context, chromePlatform *fleet.ChromePlatform) (*fleet.ChromePlatform, error) {
	return putChromePlatform(ctx, chromePlatform, false)
}

// UpdateChromePlatform updates chromePlatform in datastore.
func UpdateChromePlatform(ctx context.Context, chromePlatform *fleet.ChromePlatform) (*fleet.ChromePlatform, error) {
	return putChromePlatform(ctx, chromePlatform, true)
}

// GetChromePlatform returns chromePlatform for the given id from datastore.
func GetChromePlatform(ctx context.Context, id string) (*fleet.ChromePlatform, error) {
	pm, err := fleetds.Get(ctx, &fleet.ChromePlatform{Name: id}, newChromePlatformEntity)
	if err == nil {
		return pm.(*fleet.ChromePlatform), err
	}
	return nil, err
}

// ListChromePlatforms lists the chromePlatforms
// Does a query over ChromePlatform entities. Returns up to pageSize entities, plus non-nil cursor (if
// there are more results). pageSize must be positive.
func ListChromePlatforms(ctx context.Context, pageSize int32, pageToken string) (res []*fleet.ChromePlatform, nextPageToken string, err error) {
	q, err := fleetds.ListQuery(ctx, ChromePlatformKind, pageSize, pageToken)
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
		res = append(res, pm.(*fleet.ChromePlatform))
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
		return nil, "", status.Errorf(codes.Internal, fleetds.InternalError)
	}
	if nextCur != nil {
		nextPageToken = nextCur.String()
	}
	return
}

// DeleteChromePlatform deletes the chromePlatform in datastore
//
// For referential data intergrity,
// Delete if there are no references to the ChromePlatform by Machine/KVM in the datastore.
// If there are any references, delete will be rejected and an error message will be thrown.
func DeleteChromePlatform(ctx context.Context, id string) error {
	machines, err := registration.QueryMachineByPropertyName(ctx, "chrome_platform_id", id, true)
	if err != nil {
		return err
	}
	kvms, err := registration.QueryKVMByPropertyName(ctx, "chrome_platform_id", id, true)
	if err != nil {
		return err
	}
	if len(machines) > 0 || len(kvms) > 0 {
		var errorMsg strings.Builder
		errorMsg.WriteString(fmt.Sprintf("ChromePlatform %s cannot be deleted because there are other resources which are referring this ChromePlatform.", id))
		if len(machines) > 0 {
			errorMsg.WriteString(fmt.Sprintf("\nMachines referring the ChromPlatform:\n"))
			for _, machine := range machines {
				errorMsg.WriteString(machine.Name + ", ")
			}
		}
		if len(kvms) > 0 {
			errorMsg.WriteString(fmt.Sprintf("\nKVMs referring the ChromPlatform:\n"))
			for _, kvm := range kvms {
				errorMsg.WriteString(kvm.Name + ", ")
			}
		}
		logging.Errorf(ctx, errorMsg.String())
		return status.Errorf(codes.FailedPrecondition, errorMsg.String())
	}
	return fleetds.Delete(ctx, &fleet.ChromePlatform{Name: id}, newChromePlatformEntity)
}

// ImportChromePlatforms inserts chrome platforms to datastore.
func ImportChromePlatforms(ctx context.Context, platforms []*fleet.ChromePlatform) (*fleetds.OpResults, error) {
	protos := make([]proto.Message, len(platforms))
	utime := ptypes.TimestampNow()
	for i, p := range platforms {
		p.UpdateTime = utime
		protos[i] = p
	}
	return fleetds.Insert(ctx, protos, newChromePlatformEntity, true, true)
}

// GetAllChromePlatforms returns all platforms in record.
func GetAllChromePlatforms(ctx context.Context) (*fleetds.OpResults, error) {
	return fleetds.GetAll(ctx, queryAll)
}

func putChromePlatform(ctx context.Context, chromePlatform *fleet.ChromePlatform, update bool) (*fleet.ChromePlatform, error) {
	chromePlatform.UpdateTime = ptypes.TimestampNow()
	pm, err := fleetds.Put(ctx, chromePlatform, newChromePlatformEntity, update)
	if err == nil {
		return pm.(*fleet.ChromePlatform), err
	}
	return nil, err
}
