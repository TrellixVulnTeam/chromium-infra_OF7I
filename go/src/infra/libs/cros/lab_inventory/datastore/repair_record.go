package datastore

import (
	"context"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	inv "infra/appengine/cros/lab_inventory/api/v1"
)

// DeviceManualRepairRecordsOpRes is for use in Datastore to RPC conversions
type DeviceManualRepairRecordsOpRes struct {
	Record *inv.DeviceManualRepairRecord
	Entity *DeviceManualRepairRecordEntity
	Err    error
}

func (r *DeviceManualRepairRecordsOpRes) logError(e error) {
	r.Err = e
}

// GetDeviceManualRepairRecords returns the DeviceManualRepairRecord matching
// the device id ($hostname-$assetTag-$createdTime).
func GetDeviceManualRepairRecords(ctx context.Context, ids []string) []*DeviceManualRepairRecordsOpRes {
	queryResults := make([]*DeviceManualRepairRecordsOpRes, len(ids))
	qrMap := make(map[string]*DeviceManualRepairRecordsOpRes)
	entities := make([]*DeviceManualRepairRecordEntity, 0, len(ids))
	for _, id := range ids {
		qrMap[id] = &DeviceManualRepairRecordsOpRes{
			Entity: &DeviceManualRepairRecordEntity{
				ID: id,
			},
		}
		entities = append(entities, qrMap[id].Entity)
	}
	if err := datastore.Get(ctx, entities); err != nil {
		for i, e := range err.(errors.MultiError) {
			qrMap[entities[i].ID].logError(e)
		}
	}
	for i, id := range ids {
		queryResults[i] = qrMap[id]
	}
	return queryResults
}

// GetRepairRecordByPropertyName queries DeviceManualRepairRecord entity in the
// datastore using a given propertyName and propertyValue. Should return both
// a Record and an Entity.
func GetRepairRecordByPropertyName(ctx context.Context, propertyName, propertyValue string) ([]*DeviceManualRepairRecordsOpRes, error) {
	q := datastore.NewQuery(DeviceManualRepairRecordEntityKind)
	var entities []*DeviceManualRepairRecordEntity

	if err := datastore.GetAll(ctx, q.Eq(propertyName, propertyValue), &entities); err != nil {
		logging.Errorf(ctx, "Failed to query from datastore: %s", err)
		return nil, err
	}

	if len(entities) == 0 {
		logging.Infof(ctx, "No repair records found for the query: %s, %s", propertyName, propertyValue)
		return nil, nil
	}

	repairRecords := make([]*DeviceManualRepairRecordsOpRes, len(entities))
	for i, e := range entities {
		var r inv.DeviceManualRepairRecord
		opRes := &DeviceManualRepairRecordsOpRes{
			Entity: e,
		}
		if err := proto.Unmarshal(e.Content, &r); err != nil {
			opRes.logError(err)
		}
		opRes.Record = &r
		repairRecords[i] = opRes
	}

	return repairRecords, nil
}

// AddDeviceManualRepairRecords creates a DeviceManualRepairRecord with the
// device hostname and adds it to the datastore.
func AddDeviceManualRepairRecords(ctx context.Context, records []*inv.DeviceManualRepairRecord) ([]*DeviceManualRepairRecordsOpRes, error) {
	recLength := len(records)
	allResponses := make([]*DeviceManualRepairRecordsOpRes, recLength)
	putEntities := make([]*DeviceManualRepairRecordEntity, 0, recLength)
	putResponses := make([]*DeviceManualRepairRecordsOpRes, 0, recLength)
	var err error

	for i, r := range records {
		res := &DeviceManualRepairRecordsOpRes{
			Record: r,
		}
		allResponses[i] = res
		recordEntity, err := NewDeviceManualRepairRecordEntity(r)
		if err != nil {
			res.logError(err)
			continue
		}
		res.Entity = recordEntity

		putEntities = append(putEntities, recordEntity)
		putResponses = append(putResponses, res)
	}

	f := func(ctx context.Context) error {
		finalEntities := make([]*DeviceManualRepairRecordEntity, 0, recLength)
		finalResponses := make([]*DeviceManualRepairRecordsOpRes, 0, recLength)

		existsArr, err := deviceManualRepairRecordsExists(ctx, putEntities)
		if err == nil {
			for i, pe := range putEntities {
				_, exists := existsArr[i]
				if exists {
					putResponses[i].logError(errors.Reason("Record exists in the datastore").Err())
					continue
				}
				finalEntities = append(finalEntities, pe)
				finalResponses = append(finalResponses, putResponses[i])
			}
		} else {
			finalEntities = putEntities
			finalResponses = putResponses
		}

		if err := datastore.Put(ctx, finalEntities); err != nil {
			for i, e := range err.(errors.MultiError) {
				finalResponses[i].logError(e)
			}
		}
		return nil
	}

	err = datastore.RunInTransaction(ctx, f, nil)
	return allResponses, err
}

// UpdateDeviceManualRepairRecords updates the DeviceManualRepairRecord matching
// the device hostname in the datastore. Given a map of ids and records, it gets
// entities from the datastore first and updates the entities with the new
// record values.
func UpdateDeviceManualRepairRecords(ctx context.Context, records map[string]*inv.DeviceManualRepairRecord) ([]*DeviceManualRepairRecordsOpRes, error) {
	recLength := len(records)
	ids := make([]string, 0, recLength)
	for id := range records {
		ids = append(ids, id)
	}
	// Should catch all non-existent and empty ID record requests here
	getRecords := GetDeviceManualRepairRecords(ctx, ids)

	allResponses := make([]*DeviceManualRepairRecordsOpRes, recLength)
	putEntities := make([]*DeviceManualRepairRecordEntity, 0, recLength)
	putResponses := make([]*DeviceManualRepairRecordsOpRes, 0, recLength)
	var err error
	for i, r := range getRecords {
		res := &DeviceManualRepairRecordsOpRes{
			Record: records[r.Entity.ID],
		}
		allResponses[i] = res
		recordEntity, err := r.Entity, r.Err
		if err != nil {
			res.logError(err)
			continue
		}
		res.Entity = recordEntity

		putEntities = append(putEntities, recordEntity)
		putResponses = append(putResponses, res)
	}

	f := func(ctx context.Context) error {
		finalEntities := make([]*DeviceManualRepairRecordEntity, 0, recLength)
		finalResponses := make([]*DeviceManualRepairRecordsOpRes, 0, recLength)

		for i, pe := range putEntities {
			err := pe.UpdateDeviceManualRepairRecordEntity(putResponses[i].Record)
			if err != nil {
				putResponses[i].logError(err)
				continue
			}
			finalEntities = append(finalEntities, pe)
			finalResponses = append(finalResponses, putResponses[i])
		}

		if err := datastore.Put(ctx, finalEntities); err != nil {
			for i, e := range err.(errors.MultiError) {
				finalResponses[i].logError(e)
			}
		}
		return nil
	}

	err = datastore.RunInTransaction(ctx, f, nil)
	return allResponses, err
}

// Checks if the davice manual repair records exist in the datastore.
func deviceManualRepairRecordsExists(ctx context.Context, entities []*DeviceManualRepairRecordEntity) (map[int]bool, error) {
	existsMap := make(map[int]bool, 0)
	res, err := datastore.Exists(ctx, entities)
	if res == nil {
		return existsMap, err
	}
	for i, r := range res.List(0) {
		if r {
			existsMap[i] = true
		}
	}
	return existsMap, err
}
