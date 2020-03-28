// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package configuration

import (
	"io/ioutil"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/common/errors"

	crimsonconfig "go.chromium.org/luci/machine-db/api/config/v1"
	fleet "infra/libs/fleet/protos/go"
)

// ParsePlatformsFromFile parse chrome platforms in crimson format from local file.
func ParsePlatformsFromFile(path string) (*crimsonconfig.Platforms, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Annotate(err, "fail to read file %s", path).Err()
	}
	platforms := crimsonconfig.Platforms{}
	err = proto.UnmarshalText(string(b), &platforms)
	if err != nil {
		return nil, errors.Annotate(err, "fail to unmarshal %s", path).Err()
	}
	return &platforms, nil
}

// ToChromePlatforms converts platforms in static file to UFS format.
func ToChromePlatforms(oldP *crimsonconfig.Platforms) []*fleet.ChromePlatform {
	ps := oldP.GetPlatform()
	newP := make([]*fleet.ChromePlatform, len(ps))
	for i, p := range ps {
		newP[i] = &fleet.ChromePlatform{
			Id: &fleet.ChromePlatformID{
				Value: p.GetName(),
			},
			Manufacturer: p.GetManufacturer(),
			Description:  p.GetDescription(),
		}
	}
	return newP
}
