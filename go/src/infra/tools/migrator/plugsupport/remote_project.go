// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package plugsupport

import (
	"context"
	"infra/tools/migrator"
	"sync"

	"github.com/golang/protobuf/proto"

	"go.chromium.org/luci/common/errors"
	lucipb "go.chromium.org/luci/common/proto"
	"go.chromium.org/luci/config/cfgclient"
)

// remoteProject implements the migrator.Project interface for a remote LUCI
// project (i.e. sourced via luci-config APIs).
type remoteProject struct {
	id migrator.ReportID

	ctx context.Context

	configsOnce sync.Once
	configsErr  error
	configs     map[string]migrator.ConfigFile
}

// RemoteProject returns a new migrator.Project object for `projID`.
func RemoteProject(ctx context.Context, projID string) migrator.Project {
	return &remoteProject{
		id:  migrator.ReportID{Project: projID},
		ctx: ctx,
	}
}

func (p *remoteProject) ID() string { return p.id.Project }

func (p *remoteProject) ConfigFiles() map[string]migrator.ConfigFile {
	p.configsOnce.Do(func() {
		var files []string
		files, p.configsErr = cfgclient.Client(p.ctx).ListFiles(p.ctx, p.id.ConfigSet())
		if p.configsErr == nil {
			p.configs = make(map[string]migrator.ConfigFile, len(files))
			for _, file := range files {
				p.configs[file] = &remoteConfigFile{
					id: migrator.ReportID{
						Project:    p.id.Project,
						ConfigFile: file,
					},
					ctx: p.ctx,
				}
			}
		}
	})
	if p.configsErr != nil {
		panic(p.configsErr)
	}
	return p.configs
}

func (p *remoteProject) Report(tag, description string, opts ...migrator.ReportOption) {
	addReport(p.ctx, p.id.GenerateReport(tag, description, opts...))
}

// remoteConfigFile holds a single configuration file and its metadata.
type remoteConfigFile struct {
	id migrator.ReportID

	ctx context.Context

	rawDataOnce sync.Once
	rawDataErr  error
	rawData     string
}

func (c *remoteConfigFile) Path() string { return c.id.ConfigFile }

func (c *remoteConfigFile) RawData() string {
	c.rawDataOnce.Do(func() {
		c.rawDataErr = cfgclient.Get(
			c.ctx, c.id.ConfigSet(),
			c.id.ConfigFile, cfgclient.String(&c.rawData), nil)
	})
	if c.rawDataErr != nil {
		panic(c.rawDataErr)
	}
	return c.rawData
}

func (c *remoteConfigFile) TextPb(out proto.Message) {
	if err := lucipb.UnmarshalTextML(c.RawData(), out); err != nil {
		panic(errors.Annotate(err, "parsing TEXTPB: %s", c.id).Err())
	}
}

func (c *remoteConfigFile) Report(tag, description string, opts ...migrator.ReportOption) {
	addReport(c.ctx, c.id.GenerateReport(tag, description, opts...))
}
