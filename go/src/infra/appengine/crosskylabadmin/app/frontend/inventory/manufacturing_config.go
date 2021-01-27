// Copyright 2020 The LUCI Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package inventory

import (
	"bytes"
	"context"

	"github.com/golang/protobuf/jsonpb"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	"infra/appengine/crosskylabadmin/app/config"
	"infra/appengine/crosskylabadmin/app/gitstore"

	"go.chromium.org/chromiumos/infra/proto/go/manufacturing"
)

// GetManufacturingConfig fetch manufacturing configs from git.
func GetManufacturingConfig(ctx context.Context, gitilesC gitstore.GitilesClient) (map[string]*manufacturing.Config, error) {
	cfg := config.Get(ctx).Inventory
	gf := gitstore.FilesSpec{
		Project: cfg.ManufacturingConfigProject,
		Branch:  cfg.ManufacturingConfigBranch,
		Paths:   []string{cfg.ManufacturingConfigPath},
	}
	files, err := gitstore.FetchFiles(ctx, gitilesC, gf)
	if err != nil {
		return nil, errors.Annotate(err, "fail to fetch manufacturing configs based on %s:%s:%v", gf.Project, gf.Branch, gf.Paths).Err()
	}
	data, ok := files[cfg.ManufacturingConfigPath]
	if !ok {
		return nil, errors.Reason("no manufacturing config in path %s/%s", cfg.ManufacturingConfigProject, cfg.ManufacturingConfigPath).Err()
	}

	unmarshaler := jsonpb.Unmarshaler{AllowUnknownFields: true}
	allConfigs := manufacturing.ConfigList{}
	err = unmarshaler.Unmarshal(bytes.NewReader([]byte(data)), &allConfigs)
	if err != nil {
		return nil, errors.Annotate(err, "fail to unmarshal manufacturing config").Err()
	}
	configs := make(map[string]*manufacturing.Config, 0)
	for _, c := range allConfigs.Value {
		id := c.GetManufacturingId().GetValue()
		if _, found := configs[id]; found {
			logging.Infof(ctx, "found duplicated id: %s id")
		} else {
			configs[id] = c
		}
	}
	return configs, nil
}
