// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmds

import (
	"github.com/golang/protobuf/jsonpb"
)

// Indent is the indentation to use for JSON. Set it once for consistency.
const indent = "  "

// JsonMarshaler marshals protos to JSON using adminclient-wide settings.
var jsonMarshaler = jsonpb.Marshaler{
	Indent: indent,
}
