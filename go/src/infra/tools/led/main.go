// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// Command 'led' is the new generation of 'infra/tools/led'. The strange
// directory structure here is to allow the side-by-side existence of the old
// and new led tools.
//
// Once the old generation tool is removed, this will become the new contents of
// 'infra/tools/led'.
//
// The implementation here defers entirely to go.chromium.org/luci/led, but
// implements 'job.KitchenSupport' to facilitate working with old-style kitchen
// jobs. Once kitchen is fully deprecated, this package in infra/ will go away
// entirely, and the led CIPD package will be produced directly from
// go.chromium.org/luci/led.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"path"
	"strings"

	"infra/tools/kitchen/cookflags"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	structpb "github.com/golang/protobuf/ptypes/struct"

	bbpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/api/swarming/swarming/v1"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/led/job"
	"go.chromium.org/luci/led/ledcli"
	logdog_types "go.chromium.org/luci/logdog/common/types"
)

const bbModPropKey = "$recipe_engine/buildbucket"

type kitchenSupport struct{}

var _ job.KitchenSupport = kitchenSupport{}

func (kitchenSupport) GenerateCommand(ctx context.Context, bb *job.Buildbucket) ([]string, error) {
	kitchenArgs := cookflags.CookFlags{
		CacheDir:        bb.BbagentArgs.CacheDir,
		KnownGerritHost: bb.BbagentArgs.KnownPublicGerritHosts,
		CheckoutDir:     path.Dir(bb.BbagentArgs.ExecutablePath),

		CallUpdateBuild: false,

		SystemAccount: "system",
		TempDir:       "tmp",

		AnnotationURL: logdog_types.StreamAddr{
			Host:    bb.BbagentArgs.Build.Infra.Logdog.Hostname,
			Project: bb.BbagentArgs.Build.Infra.Logdog.Project,
			Path: (logdog_types.StreamName(bb.BbagentArgs.Build.Infra.Logdog.Prefix).
				AsPathPrefix("annotations")),
		},
	}

	if bb.FinalBuildProtoPath != "" {
		if !strings.HasSuffix(bb.FinalBuildProtoPath, ".json") {
			return nil, errors.New("FinalBuildProtoPath for kitchen must end with .json")
		}
		kitchenArgs.OutputResultJSONPath = path.Join(
			"${ISOLATED_OUTDIR}", bb.FinalBuildProtoPath)
	}

	buildCopy := proto.Clone(bb.BbagentArgs.Build).(*bbpb.Build)
	propStruct := buildCopy.Input.Properties
	buildCopy.Input.Properties = nil
	buildCopy.Infra.Buildbucket.Hostname = ""

	buildStr, err := (&jsonpb.Marshaler{}).MarshalToString(buildCopy)
	if err != nil {
		return nil, errors.Annotate(err, "serializing build").Err()
	}
	buildStr = fmt.Sprintf(`{"build": %s, "hostname": %q}`, buildStr, bb.BbagentArgs.Build.Infra.Buildbucket.Hostname)
	propStruct.Fields[bbModPropKey] = &structpb.Value{}
	err = jsonpb.UnmarshalString(buildStr, propStruct.Fields[bbModPropKey])
	if err != nil {
		return nil, errors.Annotate(err, "deserializing build").Err()
	}

	kitchenArgs.RecipeName = propStruct.Fields["recipe"].GetStringValue()

	var jsonBuf bytes.Buffer
	err = (&jsonpb.Marshaler{}).Marshal(&jsonBuf, propStruct)
	if err != nil {
		return nil, errors.Annotate(err, "serializing input properties").Err()
	}
	if err := json.Unmarshal(jsonBuf.Bytes(), &kitchenArgs.Properties); err != nil {
		return nil, errors.Annotate(err, "deserializing input properties").Err()
	}

	ret := []string{"kitchen${EXECUTABLE_SUFFIX}", "cook"}
	return append(ret, kitchenArgs.Dump()...), nil
}

func (kitchenSupport) FromSwarming(ctx context.Context, in *swarming.SwarmingRpcsNewTaskRequest, out *job.Buildbucket) error {
	ts := in.TaskSlices[0]

	var kitchenArgs cookflags.CookFlags
	fs := flag.NewFlagSet("kitchen_cook", flag.ContinueOnError)
	kitchenArgs.Register(fs)
	if err := fs.Parse(ts.Properties.Command[2:]); err != nil {
		return errors.Annotate(err, "parsing kitchen cook args").Err()
	}

	out.BbagentArgs = &bbpb.BBAgentArgs{
		CacheDir:               kitchenArgs.CacheDir,
		KnownPublicGerritHosts: ([]string)(kitchenArgs.KnownGerritHost),
		Build:                  &bbpb.Build{},

		// See note in the job.Buildbucket message for the reason we use "luciexe"
		// even in kitchen mode.
		ExecutablePath: path.Join(kitchenArgs.CheckoutDir, "luciexe"),
	}

	// kitchen builds are sorta inverted; the Build message is in the buildbucket
	// module property, but it doesn't contain the properties in input.
	bbModProps := kitchenArgs.Properties[bbModPropKey].(map[string]interface{})
	delete(kitchenArgs.Properties, bbModPropKey)

	blob, err := json.Marshal(bbModProps["build"])
	if err != nil {
		return errors.Annotate(err, "%s['build'] -> json", bbModPropKey).Err()
	}
	if err := jsonpb.Unmarshal(bytes.NewReader(blob), out.BbagentArgs.Build); err != nil {
		return errors.Annotate(err, "%s['build'] -> jsonpb", bbModPropKey).Err()
	}

	out.EnsureBasics()
	out.BbagentArgs.Build.Infra.Buildbucket.Hostname = bbModProps["hostname"].(string)

	err = jsonpb.UnmarshalString(kitchenArgs.Properties.String(),
		out.BbagentArgs.Build.Input.Properties)
	if err != nil {
		return errors.Annotate(err, "populating properties").Err()
	}

	out.WriteProperties(map[string]interface{}{
		"recipe": kitchenArgs.RecipeName,
	})
	return nil
}

func main() {
	ledcli.Main(kitchenSupport{})
}
