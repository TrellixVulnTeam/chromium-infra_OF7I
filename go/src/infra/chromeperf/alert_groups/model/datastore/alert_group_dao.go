// Copyright 2021 The Chromium Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package datastore_dao

import (
	"context"
	"time"

	"infra/chromeperf/alert_groups/model"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/gae/service/datastore"
)

type AlertGroupDAO struct {
}

func (self AlertGroupDAO) Get(ctx context.Context, id string) (*model.AlertGroup, error) {
	entity := &model.AlertGroup{ID: id}
	if err := datastore.Get(ctx, entity); err != nil {
		return nil, errors.Annotate(err, "failed to retreive object with ID %s from Datastore", id).Err()
	}
	return entity, nil
}

func (self AlertGroupDAO) Update(ctx context.Context, entity *model.AlertGroup) error {
	// Golang datastore implementation doesn't support auto updated time fields.
	// We have to update it manually.
	entity.Updated = time.Now().Round(time.Microsecond).UTC()

	if err := datastore.Put(ctx, entity); err != nil {
		return errors.Annotate(err, "failed to write entity %v to Datastore", entity).Err()
	}
	return nil
}

func (self AlertGroupDAO) Delete(ctx context.Context, entity *model.AlertGroup) error {
	if err := datastore.Delete(ctx, entity); err != nil {
		return errors.Annotate(err, "failed to delete entity %v from Datastore", entity).Err()
	}
	return nil
}
