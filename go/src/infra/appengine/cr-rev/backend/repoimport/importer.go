// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

//go:generate mockgen -source=importer.go -package repoimport -destination importer.mock.go importer

package repoimport

import (
	"context"
	"infra/appengine/cr-rev/common"
)

type importer interface {
	// Starts importing desired repository
	Run(context.Context) error
}

type importerFactory func(context.Context, common.GitRepository) importer
