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
package output

import (
	"encoding/json"
	"io"

	"go.chromium.org/luci/common/errors"
)

type benchmarkNameKey string
type storyNameKey string

type test struct {
	Actual string `json:"actual"` // Test status (PASS,FAIL)
}

type Output struct {
	Tests map[benchmarkNameKey]map[storyNameKey]test `json:"tests"`
}

func NewFromJSON(r io.Reader) (*Output, error) {
	jd := json.NewDecoder(r)
	res := Output{}
	if err := jd.Decode(&res); err != nil {
		return nil, errors.Annotate(err, "failed decoding output json").Err()
	}
	return &res, nil
}
