// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"go.chromium.org/luci/common/errors"
)

type arcTruth struct {
	Board string `json:"board"`
	Arc   string `json:"arc"`
}

func parseArc(path string) error {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return errors.Annotate(err, "load arc json %s", path).Err()
	}
	arcs := make([]arcTruth, 0)
	if err := json.Unmarshal(b, &arcs); err != nil {
		return errors.Annotate(err, "parse arc json").Err()
	}

	for _, a := range arcs {
		if a.Arc == "True" {
			fmt.Printf("\"%s\": true,\n", a.Board)
		}
	}
	return nil
}

func main() {
	if err := parseArc("./arc.json"); err != nil {
		fmt.Println(err)
	}
}
