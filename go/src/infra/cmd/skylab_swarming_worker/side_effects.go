// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/golang/protobuf/jsonpb"
	"github.com/pkg/errors"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform/side_effects"
	"infra/cmd/skylab_swarming_worker/internal/annotations"
	"infra/libs/skylab/sideeffects"
)

// dropSideEffectsConfig confirms that "content" contains a valid
// side_effects.Config JSONpb string and writes it to
// <dir>/side_effects_config.json.
func dropSideEffectsConfig(content string, dir string, logdogOutput io.Writer) (err error) {
	annotations.SeedStep(logdogOutput, "Write side_effects_config.json")
	annotations.StepCursor(logdogOutput, "Write side_effects_config.json")
	annotations.StepStarted(logdogOutput)
	defer annotations.StepClosed(logdogOutput)

	defer func() {
		if err != nil {
			annotations.StepException(logdogOutput)
			fmt.Fprint(logdogOutput, err.Error())
		}
	}()

	fmt.Fprintf(logdogOutput, "Validating side_effects.Config JSONpb:\n%s\n", content)

	var c side_effects.Config
	u := jsonpb.Unmarshaler{AllowUnknownFields: true}
	if err := u.Unmarshal(strings.NewReader(content), &c); err != nil {
		return errors.Wrap(err, "drop side_effects.Config")
	}
	if err := sideeffects.ValidateConfig(&c); err != nil {
		return errors.Wrap(err, "drop side_effects.Config")
	}
	fmt.Fprintf(logdogOutput, "Writing side_effects.Config to %s", dir)
	if err := sideeffects.WriteConfigToDisk(dir, &c); err != nil {
		return errors.Wrap(err, "drop side_effects.Config")
	}
	return nil
}
