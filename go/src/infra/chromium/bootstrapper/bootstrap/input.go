// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bootstrap

import (
	"sort"
	"strings"

	buildbucketpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/led/ledcmd"
	"go.chromium.org/luci/luciexe/exe"
	apipb "go.chromium.org/luci/swarming/proto/api"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// Input provides the relevant details from the build input to the operations
// that prepare a bootstrapped executable to run. It is safe to share a single
// instance between multiple operations that take Input.
type Input struct {
	commit          *buildbucketpb.GitilesCommit
	changes         []*buildbucketpb.GerritChange
	buildProperties *structpb.Struct
	propsProperties *BootstrapPropertiesProperties
	exeProperties   *BootstrapExeProperties
	casRecipeBundle *apipb.CASReference
}

// NewInput creates a new Input, returning an error if the build input fails to
// validate.
//
// The build input can fail to validate for the following reasons:
// * The $bootstrap/properties property is not set.
// * The $bootstrap/properties property is set, but not to a valid
//   BootstrapPropertiesProperties message.
// * The $bootstrap/exe property is not set.
// * The $bootstrap/exe property is set, but not to a valid
//   BootstrapExeProperties message.
func NewInput(build *buildbucketpb.Build) (*Input, error) {
	properties := build.GetInput().GetProperties()
	if properties == nil {
		properties = &structpb.Struct{}
	}

	// Check for the presence of required properties
	propsProperties := &BootstrapPropertiesProperties{}
	exeProperties := &BootstrapExeProperties{}
	propsToParse := map[string]interface{}{
		"$bootstrap/properties": propsProperties,
		"$bootstrap/exe":        exeProperties,
	}
	missingProps := make([]string, 0, len(propsToParse))
	for k := range propsToParse {
		if _, ok := properties.Fields[k]; !ok {
			missingProps = append(missingProps, k)
		}
	}
	if len(missingProps) != 0 {
		sort.Strings(missingProps)
		return nil, errors.Reason("the following required properties are not set: %s", strings.Join(missingProps, ", ")).Err()
	}

	casRecipeBundle := &apipb.CASReference{}
	propsToParse[ledcmd.CASRecipeBundleProperty] = casRecipeBundle

	if err := exe.ParseProperties(properties, propsToParse); err != nil {
		return nil, errors.Annotate(err, "failed to parse properties").Err()
	}

	if err := validate(propsProperties, "$bootstrap/properties"); err != nil {
		return nil, errors.Annotate(err, "failed to validate $bootstrap/properties property").Err()
	}
	if err := validate(exeProperties, "$bootstrap/exe"); err != nil {
		return nil, errors.Annotate(err, "failed to validate $bootstrap/exe property").Err()
	}

	if casRecipeBundle.Digest == nil {
		casRecipeBundle = nil
	}

	commit := proto.Clone(build.Input.GitilesCommit).(*buildbucketpb.GitilesCommit)

	changes := make([]*buildbucketpb.GerritChange, len(build.Input.GerritChanges))
	for i, change := range build.Input.GerritChanges {
		changes[i] = proto.Clone(change).(*buildbucketpb.GerritChange)
	}

	properties = proto.Clone(properties).(*structpb.Struct)
	for k := range propsToParse {
		delete(properties.Fields, k)
	}
	// TODO(gbeaty) Remove this once the builders no longer set the
	// $bootstrap property
	delete(properties.Fields, "$bootstrap")

	input := &Input{
		commit:          commit,
		changes:         changes,
		buildProperties: properties,
		propsProperties: propsProperties,
		exeProperties:   exeProperties,
		casRecipeBundle: casRecipeBundle,
	}
	return input, nil
}
