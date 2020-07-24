// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package plugsupport

import (
	"context"
	"plugin"

	"go.chromium.org/luci/common/errors"

	"infra/tools/migrator"
)

type nullImpl struct{}

var _ migrator.API = nullImpl{}

func (nullImpl) FindProblems(ctx context.Context, proj migrator.Project) {
	proj.Report("NO_REPORT_FUNCTION", "FindProblems is not defined by the plugin.")
}

func (nullImpl) ApplyFix(ctx context.Context, repo migrator.Repo) {}

// APIFromPlugin extracts all API symbols from `plug`, returning them as
// a single interface implementation.
//
// `warnings` contains a list of plugin load warnings; It would be good to
// elevate these to the user so they could be aware of e.g. a misspelled method
// name or similar.
//
// This returns an error if a plugin symbol was defined, but has the wrong type.
func APIFromPlugin(plug *plugin.Plugin) (constructor migrator.InstantiateAPI, err error) {
	sym, err := plug.Lookup("InstantiateAPI")
	if err != nil {
		return nil, errors.New(
			"plugin does not export InstantiateAPI symbol")
	}
	// NOTE: We have to try both the bare function signature and InstantiateAPI;
	// If the user writes `var InstantiateAPI plugin.InstantiateAPI = func...`,
	// then that's a different type than `func InstantiateAPI(...)`.
	factory, ok := sym.(func() migrator.API)
	if !ok {
		factory, ok = sym.(migrator.InstantiateAPI)
		if !ok {
			return nil, errors.Reason(
				"plugin exports InstantiateAPI, but has wrong type: %T", sym).Err()
		}
	}
	return func() migrator.API {
		if ret := factory(); ret != nil {
			return ret
		}
		return nullImpl{}
	}, nil
}
