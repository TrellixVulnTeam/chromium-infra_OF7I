// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package userinput

import (
	"fmt"
	"io/ioutil"
	"strings"

	"go.chromium.org/luci/common/errors"

	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
)

// SupportedLabels lists all supported labels for batch-update-duts.
const SupportedLabels = "pool, powerunit_hostname, powerunit_outlet"

// GetRequestFromFiles form requests based on user's input file.
func GetRequestFromFiles(fp string) (*fleet.BatchUpdateDutsRequest, error) {
	raw, err := ioutil.ReadFile(fp)
	if err != nil {
		return nil, err
	}
	rows := strings.Split(string(raw), "\n")
	duts := make([]*fleet.DutProperty, 0, len(rows))
	for _, r := range rows {
		if len(r) == 0 {
			fmt.Println("skip empty line")
			continue
		}
		d, err := parseRow(r)
		if err != nil {
			return nil, errors.Annotate(err, "parse input file").Err()
		}
		duts = append(duts, d)
	}
	return &fleet.BatchUpdateDutsRequest{
		DutProperties: duts,
	}, nil
}

func parseRow(row string) (*fleet.DutProperty, error) {
	cols := strings.Split(row, ",")
	if len(cols) < 2 {
		return nil, fmt.Errorf("Wrong format (%s): each row of input file should contain at least 2 columns separated by colon", row)
	}
	var d fleet.DutProperty
	hostname := cols[0]
	d.Hostname = hostname
	for _, c := range cols[1:] {
		err := parseField(c, &d)
		if err != nil {
			return nil, errors.Annotate(err, "parse row of input file").Err()
		}
	}
	return &d, nil
}

func parseField(col string, d *fleet.DutProperty) error {
	label := strings.Split(col, "=")
	if len(label) != 2 {
		return fmt.Errorf("Wrong format (%s): each column of input file should contain a label in format of name=value", col)
	}
	switch label[0] {
	case "pool":
		if d.Pool != "" {
			return fmt.Errorf("pool is already setup for host: %s", d.GetHostname())
		}
		d.Pool = label[1]
	case "powerunit_hostname":
		if d.Rpm != nil && d.Rpm.GetPowerunitHostname() != "" {
			return fmt.Errorf("powerunit_hostname is already setup for host: %s", d.GetHostname())
		}
		if d.Rpm == nil {
			d.Rpm = &fleet.DutProperty_Rpm{
				PowerunitHostname: label[1],
			}
		} else {
			d.Rpm.PowerunitHostname = label[1]
		}
	case "powerunit_outlet":
		if d.Rpm != nil && d.Rpm.GetPowerunitOutlet() != "" {
			return fmt.Errorf("powerunit_outlet is already setup for host: %s", d.GetHostname())
		}
		if d.Rpm == nil {
			d.Rpm = &fleet.DutProperty_Rpm{
				PowerunitOutlet: label[1],
			}
		} else {
			d.Rpm.PowerunitOutlet = label[1]
		}
	default:
		return fmt.Errorf("Unsupported label (%s): the supported label list is [%s]", label[0], SupportedLabels)
	}
	return nil
}
