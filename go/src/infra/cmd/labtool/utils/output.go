// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package utils

import (
	"bufio"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	fleet "infra/unifiedfleet/api/v1/proto"
)

var (
	switchTitle = []string{"Switch Name", "CapacityPort", "UpdateTime"}
)

// TimeFormat for all timestamps handled by labtools
var timeFormat = "2006-01-02-15:04:05"

// The tab writer which defines the write format
var tw = tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)

// The io writer for json output
var bw = bufio.NewWriter(os.Stdout)

// PrintProtoJSON prints the output as json
func PrintProtoJSON(pm proto.Message) {
	defer bw.Flush()
	m := jsonpb.Marshaler{
		EnumsAsInts: false,
		Indent:      "\t",
	}
	if err := m.Marshal(bw, pm); err != nil {
		fmt.Println("Failed to marshal JSON")
	}
	fmt.Println()
}

// printTitle prints the title fields in table form.
func printTitle(title []string) {
	for _, s := range title {
		fmt.Fprint(tw, fmt.Sprintf("%s\t", s))
	}
	fmt.Fprintln(tw)
}

// PrintSwitches prints the all switches in table form.
func PrintSwitches(switches []*fleet.Switch) {
	defer tw.Flush()
	printTitle(switchTitle)
	for _, s := range switches {
		printSwitch(s)
	}
}

func printSwitch(s *fleet.Switch) {
	var ts string
	if t, err := ptypes.Timestamp(s.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	//s.Name = UfleetUtil.RemovePrefix(s.Name)
	out := fmt.Sprintf("%s\t%d\t%s\t", s.GetName(), s.GetCapacityPort(), ts)
	fmt.Fprintln(tw, out)
}

// PrintSwitchesJSON prints the switch details in json format.
func PrintSwitchesJSON(switches []*fleet.Switch) {
	for _, s := range switches {
		//s.Name = UfleetUtil.RemovePrefix(s.Name)
		PrintProtoJSON(s)
	}
}
