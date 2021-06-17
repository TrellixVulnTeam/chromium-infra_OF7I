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
	"go.chromium.org/luci/gae/service/datastore"
)

// Definition of an Anomaly model.
// This is a golang copy of a corresponding type from python codebase:
// https://source.chromium.org/chromium/chromium/src/+/master:third_party/catapult/dashboard/dashboard/models/anomaly.py
type Anomaly struct {
	ID int64 `gae:"$id"`

	// We use this field to catch any properties that we don't explicitly list
	// below from Anomaly entities.If you need to use one of these fields -
	// add it to the schema.
	UndeclaredFields datastore.PropertyMap `gae:",extra"`

	Groups []*datastore.Key `gae:"groups"`
}
