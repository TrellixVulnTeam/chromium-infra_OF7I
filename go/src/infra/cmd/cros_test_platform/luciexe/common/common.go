// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package common houses common logic for all "steps" in this luciexe.
package common

import (
	"os"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/common/errors"
)

// ReadRequest is a helper to parse an arbitrary protobuf message from a file.
func ReadRequest(inFile string, request proto.Message) error {
	r, err := os.Open(inFile)
	if err != nil {
		return errors.Annotate(err, "read request").Err()
	}
	defer r.Close()
	if err := unmarshaller.Unmarshal(r, request); err != nil {
		return errors.Annotate(err, "read request").Err()
	}
	return nil
}

// WriteResponse is a helper to write an arbitrary protobuf message to a file.
func WriteResponse(outFile string, response proto.Message) error {
	w, err := os.Create(outFile)
	if err != nil {
		return errors.Annotate(err, "write response").Err()
	}
	defer w.Close()
	if err := marshaller.Marshal(w, response); err != nil {
		return errors.Annotate(err, "write response").Err()
	}
	return nil
}

var (
	unmarshaller = jsonpb.Unmarshaler{AllowUnknownFields: true}
	marshaller   = jsonpb.Marshaler{}
)
