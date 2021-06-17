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

package model

import (
	"time"

	"go.chromium.org/luci/gae/service/datastore"
)

type RevisionRange struct {
	Repository string `gae:"repository"`
	Start      int64  `gae:"start"`
	End        int64  `gae:"end"`
}

// Definition of an AlertGroup model.
// This is a golang copy of a corresponding type from python codebase:
// https://source.chromium.org/chromium/chromium/src/+/master:third_party/catapult/dashboard/dashboard/models/alert_group.py
type AlertGroup struct {
	// Identifier in form of uuid string. It corresponds to the Name
	// field in proto request definition. The model Name property is
	// unrelated.
	ID string `gae:"$id"`

	// We use this field to catch any properties that we don't explicitly list
	// below from Anomaly entities.If you need to use one of these fields -
	// add it to the schema.
	UndeclaredFields datastore.PropertyMap `gae:",extra"`

	// Name by default corresponds to test benchmark name or to a custom
	// name set through ALERT_GROUPING sparce diagnostic.
	Name             string        `gae:"name"`
	Domain           string        `gae:"domain"`
	SubscriptionName string        `gae:"subscription_name"`
	Status           int64         `gae:"status,noindex"`
	GroupType        int64         `gae:"group_type,noindex"`
	Active           bool          `gae:"active"`
	Created          time.Time     `gae:"created,noindex"`
	Updated          time.Time     `gae:"updated,noindex"`
	Revision         RevisionRange `gae:"revision"`
	ProjectId        string        `gae:"project_id"`
	// TODO(crbug.com/1218071): we should prefer int or string over Key here
	// (as well as for similar properties in other models).
	Anomalies []*datastore.Key `gae:"anomalies,noindex"`
}
