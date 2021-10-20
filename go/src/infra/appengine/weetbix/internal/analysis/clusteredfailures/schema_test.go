// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package clusteredfailures

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestSchema(t *testing.T) {
	t.Parallel()
	Convey(`With Schema`, t, func() {
		var fieldNames []string
		for _, field := range tableMetadata.Schema {
			fieldNames = append(fieldNames, field.Name)
		}
		Convey(`Time partitioning field is defined`, func() {
			for _, clusteringField := range tableMetadata.Clustering.Fields {
				So(clusteringField, ShouldBeIn, fieldNames)
			}
		})
		Convey(`Clustering fields are defined`, func() {
			for _, clusteringField := range tableMetadata.Clustering.Fields {
				So(clusteringField, ShouldBeIn, fieldNames)
			}
		})
	})
}
