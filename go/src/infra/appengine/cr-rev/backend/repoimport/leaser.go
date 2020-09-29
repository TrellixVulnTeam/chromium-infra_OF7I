// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package repoimport

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"sync"
	"time"

	"infra/appengine/cr-rev/common"
	"infra/appengine/cr-rev/models"

	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/gae/service/datastore"
)

const (
	leaseUpdateDuration = 10 * time.Minute
)

var errImportNotRequired = errors.New("the repository scan is not required")

type leaser struct {
	repo common.GitRepository
	doc  *models.Repository
}

func newLeaser(repo common.GitRepository) *leaser {
	return &leaser{
		repo: repo,
		doc: &models.Repository{
			ID: models.RepoID{
				Host:       repo.Host,
				Repository: repo.Name,
			},
		},
	}
}

// WithLease runs function f if lease can be acquired. It then periodically
// refreshes the lease. In case the lease is broken by external process, it
// cancels context passed to the function.
// Lease document will be removed iff there is error during import and lease
// was not broken.
func (l *leaser) WithLease(ctx context.Context, f func(ctx context.Context) error) error {
	err := l.acquireLease(ctx)
	if err != nil {
		return err
	}
	logging.Debugf(ctx, "Lease acquired for %s/%s", l.repo.Host, l.repo.Name)
	// Wait for go routine to finish before returning result
	wg := sync.WaitGroup{}
	wg.Add(1)

	cctx, cancel := context.WithCancel(ctx)

	cleanupOnError := true
	go func() {
		// Refresh lock periodically, and check ownership.
		defer wg.Done()
		timer := clock.NewTimer(ctx)
		timer.Reset(leaseUpdateDuration)
		for {
			select {
			case <-cctx.Done():
				return
			case <-timer.GetC():
				err := l.refreshLease(cctx)
				if err != nil {
					logging.WithError(err).Errorf(ctx, "Datastore repository state is not expected")
					cancel()
					// External process acquired the lease so we shouldn't do anything with it.
					cleanupOnError = false
					return
				}
				timer.Reset(leaseUpdateDuration)
			}
		}
	}()
	err = f(cctx)
	cancel()
	wg.Wait()

	if err != nil && cleanupOnError {
		logging.Debugf(ctx, "Releasing lease %s/%s, error: %s", l.repo.Host, l.repo.Name, err.Error())
		datastore.Delete(ctx, l.doc)
		return err
	}

	logging.Debugf(ctx, "Releasing lease %s/%s, no error", l.repo.Host, l.repo.Name)
	// Indexing is completed, so stop goroutine for refreshing lock.
	l.doc.SetIndexingCompleted(clock.Now(ctx).UTC().Round(time.Millisecond))
	return datastore.Put(ctx, l.doc)
}

// acquireLease attempts to acquire a lease which is stored in Datastore. If
// there is already an active lease, such lease may be broken only if stale
// (not renewed in deadline).
func (l *leaser) acquireLease(ctx context.Context) error {
	return datastore.RunInTransaction(ctx, func(ctx context.Context) error {
		if err := datastore.Get(ctx, l.doc); err != nil && err != datastore.ErrNoSuchEntity {
			return fmt.Errorf("error reading from datastore: %w", err)
		}
		now := clock.Now(ctx).UTC().Round(time.Millisecond)

		if !l.doc.IsScanRequired(now) {
			logging.Debugf(ctx, "the repository scan is not required (%+v)", l.doc)
			return errImportNotRequired
		}

		l.doc.SetStartIndexing(now, os.Getenv("GAE_INSTANCE"))
		if err := datastore.Put(ctx, l.doc); err != nil {
			return fmt.Errorf("error writing to datastore: %w", err)
		}
		return nil
	}, nil)
}

// refreshLease attempts to extend the lease. If the document is modified by
// external process, the lease won't be renewed and error will be returned.
func (l *leaser) refreshLease(ctx context.Context) error {
	dst := models.Repository{
		ID: l.doc.ID,
	}

	return datastore.RunInTransaction(ctx, func(ctx context.Context) error {
		if err := datastore.Get(ctx, &dst); err != nil {
			return err
		}
		if !reflect.DeepEqual(*l.doc, dst) {
			return errors.New("some other process claimed the lock, aborting import")
		}
		l.doc.ExtendLease(clock.Now(ctx).UTC().Round(time.Millisecond))
		return datastore.Put(ctx, l.doc)
	}, nil)

}
