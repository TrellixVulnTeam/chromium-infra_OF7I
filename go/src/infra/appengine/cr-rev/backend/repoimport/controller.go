// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

//go:generate mockgen -source=controller.go -package repoimport -destination controller.mock.go Controller

package repoimport

import (
	"context"

	"infra/appengine/cr-rev/common"

	"go.chromium.org/luci/common/logging"
)

// Controller is the main interface for importing entire Git repositories.
type Controller interface {
	// Index schedules import of repo and returns immediately (non blocking
	// operation). There are no guarantees if the import will be successful
	// and when will be done.
	Index(repo common.GitRepository)
	// Start continuously processes imports scheduled by Index function.
	// This function exits on context cancellation.
	Start(ctx context.Context)
}

type controller struct {
	ch              chan common.GitRepository
	importerFactory ImporterFactory
}

// NewController creates import controller that can import desired repositories
// one at the time.
func NewController(f ImporterFactory) Controller {
	// Channel should be large enough not to block when new items are added.
	// cr-rev scans about 2000 repositories and we expect that many items
	// to be in the queue only when database is completely empty.
	ch := make(chan common.GitRepository, 10000)
	return &controller{
		ch:              ch,
		importerFactory: f,
	}
}

// Index schedules import of repo and returns immediately.
func (c *controller) Index(repo common.GitRepository) {
	c.ch <- repo
}

// Start continuously processes imports scheduled by Index function. Only one
// repository will be indexed at the time.
func (c *controller) Start(ctx context.Context) {
	for {
		select {
		case repo := <-c.ch:
			importer := c.importerFactory(ctx, repo)
			err := importer.Run(ctx)
			switch err {
			case nil:
				logging.Infof(ctx, "repository %s/%s successfully imported", repo.Host, repo.Name)
			case errImportNotRequired:
				// do nothing, assume importer logged the reason why it's not required.
			default:
				logging.WithError(err).Errorf(
					ctx, "failed to import repository: %s/%s", repo.Host, repo.Name)
			}

		case <-ctx.Done():
			return
		}
	}
}
