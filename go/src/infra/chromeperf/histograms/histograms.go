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
package histograms

import (
	"encoding/json"
	"io"

	"go.chromium.org/luci/common/errors"
)

type Histogram struct {
	Name         string    `json:"name"`
	Unit         string    `json:"unit"`
	SampleValues []float64 `json:"sampleValues"`
}

func NewFromJSON(r io.Reader) ([]*Histogram, error) {
	jd := json.NewDecoder(r)
	hl := []*Histogram{}
	if err := jd.Decode(&hl); err != nil {
		return nil, errors.Annotate(err, "failed decoding histograms").Err()
	}

	// Filter out the histograms that do not have a name.
	res := []*Histogram{}
	for _, h := range hl {
		if h.Name != "" {
			res = append(res, h)
		}
	}
	return res, nil
}
