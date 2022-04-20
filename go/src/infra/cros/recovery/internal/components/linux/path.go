// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package linux

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
)

// IsPathExist checks if a given path exists or not.
// Raise error if the path does not exist.
func IsPathExist(ctx context.Context, run execs.Runner, path string) error {
	_, err := run(ctx, time.Minute, fmt.Sprintf(`test -e "%s"`, path))
	if err != nil {
		return errors.Annotate(err, "path exist").Err()
	}
	return nil
}

// SpaceType is different types of disk space used in the calculation for storage space.
type SpaceType string

const (
	SpaceTypeDisk  SpaceType = "disk"
	SpaceTypeInode SpaceType = "inodes"
)

// PathHasEnoughValue is a helper function that checks the given path's free disk space / inodes is no less than the min disk space /indoes specified.
func PathHasEnoughValue(ctx context.Context, r execs.Runner, dutName string, path string, typeOfSpace SpaceType, minSpaceNeeded float64) error {
	if err := IsPathExist(ctx, r, path); err != nil {
		return errors.Annotate(err, "path has enough value: %s: path: %q not exist", typeOfSpace, path).Err()
	}
	const mbPerGB = 1000
	var cmd string
	if typeOfSpace == SpaceTypeDisk {
		oneMB := math.Pow(10, 6)
		log.Infof(ctx, "Checking for >= %f (GB/inodes) of %s under %s on machine %s", minSpaceNeeded, typeOfSpace, path, dutName)
		cmd = fmt.Sprintf(`df -PB %.f %s | tail -1`, oneMB, path)
	} else {
		// checking typeOfSpace == "inodes"
		cmd = fmt.Sprintf(`df -Pi %s | tail -1`, path)
	}
	output, err := r(ctx, time.Minute, cmd)
	if err != nil {
		return errors.Annotate(err, "path has enough value: %s", typeOfSpace).Err()
	}
	outputList := strings.Fields(output)
	free, err := strconv.ParseFloat(outputList[3], 64)
	if err != nil {
		log.Errorf(ctx, err.Error())
		return errors.Annotate(err, "path has enough value: %s", typeOfSpace).Err()
	}
	if typeOfSpace == SpaceTypeDisk {
		free = float64(free) / mbPerGB
	}
	if free < minSpaceNeeded {
		return errors.Reason("path has enough value: %s: Not enough free %s on %s - %f (GB/inodes) free, want %f (GB/inodes)", typeOfSpace, typeOfSpace, path, free, minSpaceNeeded).Err()
	}
	log.Infof(ctx, "Found %f (GB/inodes) >= %f (GB/inodes) of %s under %s on machine %s", free, minSpaceNeeded, typeOfSpace, path, dutName)
	return nil
}

// PathOccupiedSpacePercentage will find the percentage indicating the occupied space under the specified path.
func PathOccupiedSpacePercentage(ctx context.Context, r execs.Runner, path string) (float64, error) {
	if err := IsPathExist(ctx, r, path); err != nil {
		return -1, errors.Annotate(err, "path occupied space percentage: path: %q not exist", path).Err()
	}
	cmd := fmt.Sprintf(`df %s | tail -1`, path)
	output, err := r(ctx, time.Minute, cmd)
	if err != nil {
		return -1, errors.Annotate(err, "path occupied space percentage").Err()
	}
	// The 5th element is the percentage value of the free disk space for this path.
	outputList := strings.Fields(output)
	percentageString := strings.TrimRight(outputList[4], "%")
	occupied, err := strconv.ParseFloat(percentageString, 64)
	if err != nil {
		log.Errorf(ctx, err.Error())
		return -1, errors.Annotate(err, "path occupied space percentage").Err()
	}
	log.Infof(ctx, "Found %v%% occupied space under %s", occupied, path)
	return occupied, nil
}
