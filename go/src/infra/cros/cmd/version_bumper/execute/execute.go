// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package execute

import (
	"context"
	"fmt"
	"os"

	"infra/cros/internal/chromeosversion"

	vpb "go.chromium.org/chromiumos/infra/proto/go/chromiumos/version_bumper"
	"go.chromium.org/luci/common/errors"
)

func validate(input *vpb.BumpVersionRequest) error {
	if input.ChromiumosOverlayRepo == "" {
		return fmt.Errorf("chomiumosOverlayRepo required")
	} else if _, err := os.Stat(input.ChromiumosOverlayRepo); err != nil {
		return errors.Annotate(err, "%s could not be found", input.ChromiumosOverlayRepo).Err()
	}
	if input.ComponentToBump == vpb.BumpVersionRequest_COMPONENT_TYPE_UNSPECIFIED {
		return fmt.Errorf("componentToBump was unspecified")
	}
	return nil
}

func componentToBump(input *vpb.BumpVersionRequest) (chromeosversion.VersionComponent, error) {
	switch component := input.ComponentToBump; component {
	case vpb.BumpVersionRequest_COMPONENT_TYPE_MILESTONE:
		return chromeosversion.ChromeBranch, nil
	case vpb.BumpVersionRequest_COMPONENT_TYPE_BUILD:
		return chromeosversion.Build, nil
	case vpb.BumpVersionRequest_COMPONENT_TYPE_BRANCH:
		return chromeosversion.Branch, nil
	case vpb.BumpVersionRequest_COMPONENT_TYPE_PATCH:
		return chromeosversion.Patch, nil
	default:
		return chromeosversion.Unspecified, fmt.Errorf("bad/unspecified version component")
	}
}

// Run executes the core logic for version_bumper.
func Run(ctx context.Context, input *vpb.BumpVersionRequest) error {
	if err := validate(input); err != nil {
		return err
	}

	vinfo, err := chromeosversion.GetVersionInfoFromRepo(input.ChromiumosOverlayRepo)
	if err != nil {
		return errors.Annotate(err, "error getting version info from repo").Err()
	}

	component, _ := componentToBump(input)
	vinfo.IncrementVersion(component)

	if err := vinfo.UpdateVersionFile(); err != nil {
		return errors.Annotate(err, "error updating version file").Err()
	}

	return nil
}
