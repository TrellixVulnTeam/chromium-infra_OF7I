// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bootstrap

import (
	buildbucketpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/luciexe/exe"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// Input provides the relevant details from the build input to the operations
// that prepare a bootstrapped executable to run. It is safe to share a single
// instance between multiple operations that take Input.
type Input struct {
	commit          *buildbucketpb.GitilesCommit
	buildProperties *structpb.Struct
	properties      *BootstrapProperties
}

// NewInput creates a new Input, returning an error if the $bootstrap property
// on the build is missing or invalid.
func NewInput(build *buildbucketpb.Build) (*Input, error) {
	bootstrapProperties := &BootstrapProperties{}
	properties := build.GetInput().GetProperties()
	if properties == nil {
		properties = &structpb.Struct{}
	}
	if err := exe.ParseProperties(properties, map[string]interface{}{
		"$bootstrap": bootstrapProperties,
	}); err != nil {
		return nil, errors.Annotate(err, "failed to parse $bootstrap property").Err()
	}

	if err := validate(bootstrapProperties, "$bootstrap"); err != nil {
		return nil, errors.Annotate(err, "failed to validate $bootstrap property").Err()
	}

	properties = proto.Clone(properties).(*structpb.Struct)
	delete(properties.Fields, "$bootstrap")

	input := &Input{
		commit:          proto.Clone(build.Input.GitilesCommit).(*buildbucketpb.GitilesCommit),
		buildProperties: properties,
		properties:      bootstrapProperties,
	}
	return input, nil
}
