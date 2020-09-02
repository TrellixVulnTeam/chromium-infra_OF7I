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
	importerFactory importerFactory
}

// NewController creates import controller that can import desired repositories
// one at the time.
func NewController(f importerFactory) Controller {
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
			if err != nil {
				logging.WithError(err).Errorf(ctx, "failed to import repository")
			}

		case <-ctx.Done():
			return
		}
	}
}
