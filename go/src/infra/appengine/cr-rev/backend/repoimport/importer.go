// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

//go:generate mockgen -source=importer.go -package repoimport -destination importer.mock.go importer

package repoimport

import (
	"context"
	"infra/appengine/cr-rev/common"
)

// Importer defines interface for importing a repository.
type Importer interface {
	// Run starts importing desired repository.
	Run(context.Context) error
}

// ImporterFactory returns Importer for given repository.
type ImporterFactory func(context.Context, common.GitRepository) Importer
