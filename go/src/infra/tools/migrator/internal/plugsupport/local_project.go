// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package plugsupport

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/golang/protobuf/proto"

	"go.chromium.org/luci/common/errors"
	lucipb "go.chromium.org/luci/common/proto"
	configpb "go.chromium.org/luci/common/proto/config"

	"infra/tools/migrator"
)

type localProject struct {
	id   migrator.ReportID
	repo *repo
	ctx  context.Context

	relConfigRoot          string
	relGeneratedConfigRoot string

	configsOnce sync.Once
	configsErr  error
	configs     map[string]migrator.ConfigFile
}

var _ migrator.Project = (*localProject)(nil)

func (l *localProject) ID() string { return l.id.Project }

func (l *localProject) ConfigFiles() map[string]migrator.ConfigFile {
	dir := filepath.Join(l.repo.root, l.relGeneratedConfigRoot)

	l.configsOnce.Do(func() {
		l.configs = make(map[string]migrator.ConfigFile)
		l.configsErr = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() && filepath.Base(path) == ".git" {
				return filepath.SkipDir
			}
			if info.Mode().IsRegular() {
				relpath := filepath.ToSlash(path[len(dir)+1:])
				l.configs[relpath] = &localConfigFile{
					id: migrator.ReportID{
						Checkout:   l.id.Checkout,
						Project:    l.id.Project,
						ConfigFile: relpath,
					},
					abs: path,
					ctx: l.ctx,
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
	getReportSink(l.ctx).add(l.id, tag, description, opts...)
}

func (l *localProject) ConfigRoot() string          { return "/" + l.relConfigRoot }
func (l *localProject) GeneratedConfigRoot() string { return "/" + l.relGeneratedConfigRoot }
func (l *localProject) Repo() migrator.Repo         { return l.repo }

func (l *localProject) Shell() migrator.Shell {
	return &shell{
		ctx:  l.ctx,
		root: l.repo.root,
		cwd:  l.relConfigRoot,
	}
}

func (l *localProject) RegenerateConfigs() {
	// Attempt to read lucicfg invocation details from project.cfg.
	f := l.ConfigFiles()["project.cfg"]
	if f == nil {
		panic(errors.Reason("no project.cfg in the configs").Err())
	}
	var cfg configpb.ProjectCfg
	f.TextPb(&cfg)

	// Use the config metadata, if available, or "guess" main.star.
	meta := cfg.GetLucicfg()
	if meta.GetEntryPoint() == "" {
		meta = &configpb.GeneratorMetadata{
			EntryPoint: "main.star",
		}
	}

	// Sort vars for less random logging output.
	vars := make([]string, 0, len(meta.Vars))
	for k, v := range meta.Vars {
		vars = append(vars, fmt.Sprintf("%s=%s", k, v))
	}
	sort.Strings(vars)
	cmd := []string{"generate", meta.EntryPoint}
	for _, v := range vars {
		cmd = append(cmd, "-var", v)
	}

	// lucicfg logs to stderr, make its output less red in the migrator logs.
	cmd = append(cmd, migrator.TieStderr)

	l.Shell().Run("lucicfg", cmd...)
}

type localConfigFile struct {
	id  migrator.ReportID
	abs string
	ctx context.Context

	rawDataOnce sync.Once
	rawDataErr  error
	rawData     string
}

func (l *localConfigFile) Path() string { return l.id.ConfigFile }

func (l *localConfigFile) RawData() string {
	l.rawDataOnce.Do(func() {
		data, err := ioutil.ReadFile(l.abs)
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
	getReportSink(l.ctx).add(l.id, tag, description, opts...)
}
