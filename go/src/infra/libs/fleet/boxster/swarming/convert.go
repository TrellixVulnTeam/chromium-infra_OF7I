// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package swarming

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/PaesslerAG/jsonpath"
	"github.com/golang/protobuf/jsonpb"
	"go.chromium.org/chromiumos/config/go/payload"
	"go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/luci/common/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	nilSelectErr  = "unsupported value type <nil> for select"
	invalidKeyErr = "could not select value, invalid key"
)

var labelMarshaler = jsonpb.Marshaler{
	EnumsAsInts:  false,
	EmitDefaults: true,
	Indent:       "  ",
	OrigName:     true,
}

// ConvertAll converts one DutAttribute label to multiple Swarming labels.
//
// The converted labels are returned in the form of `${label_prefix}:val1,val2`
// in an array. Each label value is comma-separated. Label prefixes are the
// DutAttribute ID and the aliases listed.
func ConvertAll(dutAttr *api.DutAttribute, bundle *payload.FlatConfig) ([]string, error) {
	var prefixes []string
	labelPrefix := dutAttr.GetId().GetValue()
	if labelPrefix == "" {
		return prefixes, status.Errorf(codes.Internal, "DutAttribute has no ID")
	}
	prefixes = append(prefixes, labelPrefix)

	aliases := dutAttr.GetAliases()
	prefixes = append(prefixes, aliases...)

	var labels []string
	for _, p := range prefixes {
		v, err := convertSingle(p, dutAttr, bundle)
		if err != nil || v == "" {
			continue
		}
		labels = append(labels, v)
	}

	return labels, nil
}

// convertSingle converts one DutAttribute label to one Swarming label.
//
// The converted label is returned in the form of `${label_prefix}:val1,val2`.
// Each label value is comma-separated.
func convertSingle(labelPrefix string, dutAttr *api.DutAttribute, bundle *payload.FlatConfig) (string, error) {
	var values string
	var err error

	// Construct and try each path defined in DutAttribute. Tried in order. First
	// path to return a value will be used.
	jsonPaths, err := ConstructJsonPaths(dutAttr)
	if err != nil {
		return "", err
	}
	for _, p := range jsonPaths {
		values, err = getLabelValues(p, bundle)
		if err != nil || values == "" {
			continue
		}
		break
	}

	// Exhausted all possible paths defined in DutAttribute. If values is empty,
	// then no values found.
	if values == "" {
		return "", status.Errorf(codes.NotFound, "No supported config source found")
	}
	return fmt.Sprintf("%s:%s", labelPrefix, values), nil
}

// getLabelValues takes a path and returns the corresponding FlatConfig value.
//
// It uses a jsonpath string to try to find corresponding values in a
// FlatConfig. It returns a comma-separated string of the values found.
func getLabelValues(jsonGetPath string, bundle *payload.FlatConfig) (string, error) {
	js, err := labelMarshaler.MarshalToString(bundle)
	if err != nil {
		return "", err
	}

	fc := interface{}(nil)
	err = json.Unmarshal([]byte(js), &fc)
	if err != nil {
		return "", err
	}

	labelVals, err := jsonpath.Get(jsonGetPath, fc)
	if err != nil {
		return "", err
	}
	return constructLabelValuesString(labelVals), nil
}

// constructLabelValuesString takes label values and returns them as a string.
//
// It takes an interface of label values parsed from a json object and returns a
// comma-separated string of the values. The interfaces supported are primitive
// types and iterable interfaces.
func constructLabelValuesString(labelVals interface{}) string {
	var rsp string
	switch x := labelVals.(type) {
	case []interface{}:
		valsArr := []string{}
		for _, i := range x {
			valsArr = append(valsArr, i.(string))
		}
		rsp = strings.Join(valsArr, ",")
	default:
		rsp = labelVals.(string)
	}
	return rsp
}

// ConstructJsonPaths returns config paths defined by a DutAttribute.
//
// It takes a DutAttribute and returns an array of field paths defined in
// jsonpath syntax. The sources that are currently supported are:
//   1. FlatConfigSource
//   2. HwidSource
func ConstructJsonPaths(dutAttr *api.DutAttribute) ([]string, error) {
	if dutAttr.GetFlatConfigSource() != nil {
		return generateFlatConfigSourcePaths(dutAttr), nil
	} else if dutAttr.GetHwidSource() != nil {
		return generateHwidSourcePaths(dutAttr), nil
	}
	return []string{}, errors.Reason("No supported config source found").Err()
}

// generateFlatConfigSourcePaths returns config paths defined by a DutAttribute.
//
// It takes a DutAttribute and returns an array of FlatConfigSource field paths
// strings defined in jsonpath syntax.
func generateFlatConfigSourcePaths(dutAttr *api.DutAttribute) []string {
	var rsp []string
	for _, f := range dutAttr.GetFlatConfigSource().GetFields() {
		rsp = append(rsp, fmt.Sprintf("$.%s", f.GetPath()))
	}
	return rsp
}

// generateHwidSourcePaths returns config paths defined by a DutAttribute.
//
// It takes a DutAttribute and returns an array of HwidSource field paths
// strings defined in jsonpath syntax.
func generateHwidSourcePaths(dutAttr *api.DutAttribute) []string {
	var rsp []string
	componentType := dutAttr.GetHwidSource().GetComponentType()
	for _, f := range dutAttr.GetHwidSource().GetFields() {
		rsp = append(rsp, fmt.Sprintf(`$.hw_components[?(@.%s != null)].%s`, componentType, f.GetPath()))
	}
	return rsp
}
