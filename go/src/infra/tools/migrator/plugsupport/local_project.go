// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package plugsupport

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/golang/protobuf/proto"

	"go.chromium.org/luci/common/errors"
	lucipb "go.chromium.org/luci/common/proto"

	"infra/tools/migrator"
)

type localProject struct {
	id  migrator.ReportID
	dir string
	ctx context.Context

	configsOnce sync.Once
	configsErr  error
	configs     map[string]migrator.ConfigFile
}

var _ migrator.Project = (*localProject)(nil)

func (l *localProject) ID() string { return l.id.Project }

func (l *localProject) ConfigFiles() map[string]migrator.ConfigFile {
	l.configsOnce.Do(func() {
		l.configsErr = filepath.Walk(l.dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() && filepath.Base(path) == ".git" {
				return filepath.SkipDir
			}
			if info.Mode().IsRegular() {
				relpath := path[len(l.dir)+1:]
				l.configs[relpath] = &localConfigFile{
					id: migrator.ReportID{
						Project:    l.id.Project,
						ConfigFile: relpath,
					},
					generatedConfigRoot: l.dir,
					ctx:                 l.ctx,
				}
			}
			return nil
		})
	})
	if l.configsErr != nil {
		panic(l.configsErr)
	}
	return l.configs
}

func (l *localProject) Report(tag, description string, opts ...migrator.ReportOption) {
	addReport(l.ctx, l.id.GenerateReport(tag, description, opts...))
}

type localConfigFile struct {
	id migrator.ReportID

	generatedConfigRoot string

	ctx context.Context

	rawDataOnce sync.Once
	rawDataErr  error
	rawData     string
}

func (l *localConfigFile) Path() string { return l.id.ConfigFile }

func (l *localConfigFile) RawData() string {
	l.rawDataOnce.Do(func() {
		data, err := ioutil.ReadFile(filepath.Join(l.generatedConfigRoot, l.id.ConfigFile))
		l.rawData = string(data)
		l.rawDataErr = err
	})
	if l.rawDataErr != nil {
		panic(l.rawDataErr)
	}
	return l.rawData
}

func (l *localConfigFile) TextPb(out proto.Message) {
	if err := lucipb.UnmarshalTextML(l.RawData(), out); err != nil {
		panic(errors.Annotate(err, "parsing TEXTPB: %s", l.id).Err())
	}
}

func (l *localConfigFile) Report(tag, description string, opts ...migrator.ReportOption) {
	addReport(l.ctx, l.id.GenerateReport(tag, description, opts...))
}
