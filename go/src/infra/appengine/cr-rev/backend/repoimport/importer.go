// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package repoimport

import (
	"context"
	"infra/appengine/cr-rev/common"
)

var importerKey = "repoimport.Importer"

type importer interface {
	// Starts importing desired repository
	Run(context.Context) error
}

type importerFactory func(context.Context, common.GitRepository) importer

// getImporter returns a new instance of importer. If context doesn't have
// override value set, it defaults to Gitiles Importer.
func getImporter(ctx context.Context, repo common.GitRepository) importer {
	if v := ctx.Value(&importerKey); v != nil {
		return v.(importerFactory)(ctx, repo)
	}
	return newGitilesImporter(ctx, repo)
}

// setImporterFactory stores importerFactory into context. Useful for
// overriding default behavior in unit tests.
func setImporterFactory(ctx context.Context, f importerFactory) context.Context {
	return context.WithValue(ctx, &importerKey, f)
}
