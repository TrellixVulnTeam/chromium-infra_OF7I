// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	kartepb "infra/cros/karte/api"
)

// ActionKind is the kind of an action
const ActionKind = "ActionKind"

// ActionEntity is the datastore entity for actions.
type ActionEntity struct {
	_kind string `gae:"$kind,ActionKind"`
	ID    string `gae:"$id"`
	Proto []byte `gae:"proto"`
}

// ConvertActionEntityToAction converts a datastore action entity to an action proto.
func ConvertActionEntityToAction(e *ActionEntity) *kartepb.Action {
	if e == nil {
		panic("action entity cannot be nil")
	}
	return &kartepb.Action{
		Name: e.ID,
	}
}

// ConvertActionEntitiesToActions converts a slice of datastore action entities to a slice of action protos.
func ConvertActionEntitiesToActions(es ...*ActionEntity) []*kartepb.Action {
	var actions []*kartepb.Action
	for _, e := range es {
		actions = append(actions, ConvertActionEntityToAction(e))
	}
	return actions
}

// ConvertObservationEntityToObservation converts a datastore observation entity to an observation proto.
// TODO(gregorynisbet): replace with type once defined.
func ConvertObservationEntityToObservation(e interface{}) *kartepb.Observation {
	panic("not implemented")
}

// ConvertObservationEntitiesToObservations converts a slice of datastore observation entities to a slice of observation protos.
// TODO(gregorynisbet): replace with type once defined.
func ConvertObservationEntitiesToObservations(es interface{}) []*kartepb.Observation {
	panic("not implemented")
}
