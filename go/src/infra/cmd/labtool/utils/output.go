// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package utils

import (
	"bufio"
	"fmt"
	"os"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
)

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
