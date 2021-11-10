// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package swarming

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/golang/protobuf/jsonpb"
	"github.com/google/go-cmp/cmp"
	"go.chromium.org/chromiumos/config/go/payload"
	"go.chromium.org/chromiumos/config/go/test/api"
)

// TODO (justinsuen): jsonpb throws an error when working with the
// publicReplication field. "json: cannot unmarshal string into Go value of type
// map[string]json.RawMessage." The field is removed just for the unittests here
// but should not affect the functionality of the library as the protos will be
// directly unmarshaled from the datastore instead of a string representation.

func parseDutAttribute(t *testing.T, protoText string) api.DutAttribute {
	var da api.DutAttribute
	if err := jsonpb.UnmarshalString(protoText, &da); err != nil {
		t.Fatalf("Error unmarshalling example text: %s", err)
	}
	return da
}

func TestConvertAll(t *testing.T) {
	t.Parallel()

	b, err := ioutil.ReadFile("test_flat_config.cfg")
	if err != nil {
		t.Fatalf("Error reading test FlatConfig: %s", err)
	}

	var fc payload.FlatConfig
	unmarshaller := &jsonpb.Unmarshaler{AllowUnknownFields: false}
	if err = unmarshaller.Unmarshal(bytes.NewBuffer(b), &fc); err != nil {
		t.Fatalf("Error unmarshalling test FlatConfig: %s", err)
	}

	t.Run("convert label with existing correct field path - single value", func(t *testing.T) {
		daText := `{
			"id": {
				"value": "attr-design"
			},
			"aliases": [
				"attr-model",
				"label-model"
			],
			"flatConfigSource": {
				"fields": [
					{
						"path": "hw_design.id.value"
					}
				]
			}
		}`
		da := parseDutAttribute(t, daText)
		want := []string{"attr-design:Test", "attr-model:Test", "label-model:Test"}
		got, err := ConvertAll(&da, &fc)
		if err != nil {
			t.Fatalf("ConvertAll failed: %s", err)
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("ConvertAll returned unexpected diff (-want +got):\n%s", diff)
		}
	})

	t.Run("convert label with existing correct field path - no matching values", func(t *testing.T) {
		daText := `{
      "id": {
        "value": "attr-ec-type"
      },
      "aliases": [
        "label-ec_type"
      ],
      "flatConfigSource": {
        "fields": [
          {
            "path": "hw_design_config.hardware_features.embedded_controller.ec_type"
          }
        ]
      }
    }`
		da := parseDutAttribute(t, daText)
		got, err := ConvertAll(&da, &fc)
		if err != nil {
			t.Fatalf("ConvertAll failed: %s", err)
		}
		if diff := cmp.Diff([]string(nil), got); diff != "" {
			t.Errorf("ConvertAll returned unexpected diff (-want +got):\n%s", diff)
		}
	})

	t.Run("convert label with existing correct field path - filter based on component", func(t *testing.T) {
		daText := `{
			"id": {
				"value": "hw-wireless"
			},
			"aliases": [
				"label-wifi_chip"
			],
			"hwidSource": {
				"componentType": "wifi",
				"fields": [
					{
						"path": "hwid_label"
					}
				]
			}
		}`
		da := parseDutAttribute(t, daText)
		want := []string{"hw-wireless:wireless_test1", "label-wifi_chip:wireless_test1"}
		got, err := ConvertAll(&da, &fc)
		if err != nil {
			t.Fatalf("ConvertAll failed: %s", err)
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("ConvertAll returned unexpected diff (-want +got):\n%s", diff)
		}
	})

	t.Run("convert label with existing correct field path - filter based on component; array of values", func(t *testing.T) {
		daText := `{
			"id": {
				"value": "hw-storage"
			},
			"aliases": [
				"label-storage"
			],
			"hwidSource": {
				"componentType": "storage",
				"fields": [
					{
						"path": "hwid_label"
					}
				]
			}
		}`
		da := parseDutAttribute(t, daText)
		want := []string{
			"hw-storage:storage_test1,storage_test2,storage_test3",
			"label-storage:storage_test1,storage_test2,storage_test3",
		}
		got, err := ConvertAll(&da, &fc)
		if err != nil {
			t.Fatalf("ConvertAll failed: %s", err)
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("ConvertAll returned unexpected diff (-want +got):\n%s", diff)
		}
	})

	t.Run("convert label with non-existent field path", func(t *testing.T) {
		daText := `{
			"id": {
				"value": "attr-test"
			},
			"aliases": [
				"label-test"
			],
			"flatConfigSource": {
				"fields": [
					{
						"path": "test.attr.id.value"
					}
				]
			}
		}`
		da := parseDutAttribute(t, daText)
		got, err := ConvertAll(&da, &fc)
		if err != nil {
			t.Fatalf("ConvertAll failed: %s", err)
		}
		if diff := cmp.Diff([]string(nil), got); diff != "" {
			t.Errorf("ConvertAll returned unexpected diff (-want +got):\n%s", diff)
		}
	})
}
