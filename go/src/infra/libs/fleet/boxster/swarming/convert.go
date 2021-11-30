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

var LabelMarshaler = jsonpb.Marshaler{
	EnumsAsInts:  false,
	EmitDefaults: true,
	Indent:       "  ",
	OrigName:     true,
}

// ConvertAll converts one DutAttribute label to multiple Swarming labels.
//
// The converted labels are returned in the form of `${label_name}:val1,val2`
// in an array. Each label value is comma-separated. Label labelNames are the
// DutAttribute ID and the aliases listed.
func ConvertAll(dutAttr *api.DutAttribute, flatConfig *payload.FlatConfig) ([]string, error) {
	labelNames, err := GetLabelNames(dutAttr)
	if err != nil {
		return nil, err
	}

	// Construct and try each path defined in DutAttribute. Tried in order. First
	// path to return a value will be used.
	jsonPaths, err := ConstructJsonPaths(dutAttr)
	if err != nil {
		return nil, err
	}

	for _, p := range jsonPaths {
		valuesStr, err := GetFlatConfigLabelValuesStr(p, flatConfig)
		if err == nil && valuesStr != "" {
			return formLabels(labelNames, valuesStr)
		}
	}
	return nil, errors.Reason("No supported config source found").Err()
}

// formLabels pairs label names with the label values `${label_name}:val1,val2`.
func formLabels(labelNames []string, valuesStr string) ([]string, error) {
	// Exhausted all possible paths defined in DutAttribute. If valuesStr is empty,
	// then no values found.
	if valuesStr == "" {
		return nil, status.Errorf(codes.NotFound, "No label values found in config source found")
	}

	var labels []string
	for _, n := range labelNames {
		labels = append(labels, fmt.Sprintf("%s:%s", n, valuesStr))
	}
	if len(labels) == 0 {
		return nil, errors.New("No labels can be generated")
	}
	return labels, nil
}

// GetLabelNames extracts all possible label names from a DutAttribute.
//
// For each DutAttribute, the main label name is defined by its ID value. In
// addition, users can define other aliases. GetLabelNames will return all as
// valid label names.
func GetLabelNames(dutAttr *api.DutAttribute) ([]string, error) {
	name := dutAttr.GetId().GetValue()
	if name == "" {
		return nil, errors.New("DutAttribute has no ID")
	}
	return append([]string{name}, dutAttr.GetAliases()...), nil
}

// GetFlatConfigLabelValuesStr takes a path and returns the FlatConfig value.
//
// It uses a jsonpath string to try to find corresponding values in a
// FlatConfig. It returns a comma-separated string of the values found.
func GetFlatConfigLabelValuesStr(jsonGetPath string, flatConfig *payload.FlatConfig) (string, error) {
	js, err := LabelMarshaler.MarshalToString(flatConfig)
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
	return ConstructLabelValuesString(labelVals), nil
}

// ConstructLabelValuesString takes label values and returns them as a string.
//
// It takes an interface of label values parsed from a json object and returns a
// comma-separated string of the values. The interfaces supported are primitive
// types and iterable interfaces.
func ConstructLabelValuesString(labelVals interface{}) string {
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
