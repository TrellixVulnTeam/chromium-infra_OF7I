// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package idstrategy

import (
	"context"
	"fmt"
	"time"

	"go.chromium.org/luci/common/clock"

	kartepb "infra/cros/karte/api"
	"infra/cros/karte/internal/errors"
	"infra/cros/karte/internal/scalars"
	"infra/cros/karte/internal/uuid"
)

// Key is an opaque key type for storing things in the context.
type key string

// StrategyKey is the key for the strategy in the context.
const strategyKey = key("strategy key")

// Get gets the current strategy from the context.
func Get(ctx context.Context) Strategy {
	strategy := ctx.Value(strategyKey)
	if strategy == nil {
		panic("strategy from context is unexpectedly nil")
	}
	return strategy.(Strategy)
}

// Use produces a new context with the given ID generation strategy as the strategy.
func Use(ctx context.Context, strategy Strategy) context.Context {
	return context.WithValue(ctx, strategyKey, strategy)
}

// Strategy controls how to convert an entity or record into a UUID that is used as a datastore key.
type Strategy interface {
	// IDForAction takes an action and returns an ID.
	IDForAction(ctx context.Context, action *kartepb.Action) (string, error)

	// IDForObservation takes an observation and returns an ID.
	IDForObservation(ctx context.Context, observation *kartepb.Observation) (string, error)
}

// ProdStrategy generates an ID that takes into account the time that an entity was created and appends a UUID for disambiguation.
type prodStrategy struct{}

// IDForAction takes an action and generates an ID.
func (s *prodStrategy) IDForAction(ctx context.Context, action *kartepb.Action) (string, error) {
	// Here we use the action create time given to us in the request proto instead of time.Now() so that
	// It is possible to backfill Karte with data from past tasks.
	// We don't trust these timestamps completely (after all, backfilled timestamps are lies), but the UUID suffix
	// should do a good job of guaranteeing uniqueness.
	// Additionally, Karte queries depend on the end_time of the event *as reported by the event*.
	// Events also have an a priori maximum duration,  which means that we can perform a semantically correct query based on the
	// end time using IDs whose lexicographic sort order takes the current timestamp into account.
	msg, err := makeID(ctx, scalars.ConvertTimestampPtrToTime(action.GetCreateTime()))
	return msg, err
}

// IDForObservation takes an action and generates an ID.
func (s *prodStrategy) IDForObservation(ctx context.Context, observation *kartepb.Observation) (string, error) {
	msg, err := makeID(ctx, clock.Now(ctx))
	return msg, err
}

// NewDefault instantiates the default strategy, which is the production strategy.
func NewDefault() Strategy {
	return &prodStrategy{}
}

// NaiveStrategy produces incremental IDs in a naive, non-threadsafe way. It is useful only for tests.
type naiveStrategy struct {
	counter int64
}

// IDForAction returns entityn where n is the next number in sequence.
func (s *naiveStrategy) IDForAction(_ context.Context, _ *kartepb.Action) (string, error) {
	out := fmt.Sprintf("entity%d", s.counter)
	s.counter += 1
	return out, nil
}

// IDForObservation returns entityn where n is the next number in sequence.
func (s *naiveStrategy) IDForObservation(_ context.Context, _ *kartepb.Observation) (string, error) {
	out := fmt.Sprintf("entity%d", s.counter)
	s.counter += 1
	return out, nil
}

// NewNaive creates a new naive strategy.
func NewNaive() Strategy {
	return &naiveStrategy{counter: 1}
}

// This version number identifies the format used for IDs.
// If you are making an incompatible change, increment the version used here.
// Since the version is the first number in the ID, this means that future ID versions will necessary sort
// after older ID versions.
//
// 001 has the form 001-%021d-%010d-%s
//                  version - unix seconds - nanoseconds since beginning of second - uuid disambiguation suffix.
//
const idSchemeVersion = 1

// MakeID makes an unambiguous ID for a given entity.
func makeID(ctx context.Context, t time.Time) (string, error) {
	suffix, err := uuid.UUID(ctx)
	if err != nil {
		return "", errors.Annotate(err, "make id").Err()
	}
	return makeRawID(t, suffix)
}

// IDFmt describes how to build an ID string.
// Maximum value for an int64 is 9,223,372,036,854,775,807.
// This is 19 bits wide. Use 21 bits for safety.
// A nanosecond is one second * 10^(-9). Therefore, the maximum value
// that the nanosecond part can have is 9 digits long.
// Use 10 for safety.
const idFmt = "%03d-%021d-%010d-%s"

// MakeRawID makes an ID for a given entity by taking a time (the creation or ingestion time, depending on the kind).
// The uuidSuffix is a uuid that will be used as a disambiguation suffix.
func makeRawID(t time.Time, uuidSuffix string) (string, error) {
	if t.IsZero() {
		return "", errors.New("make id: timestamp is zero")
	}
	if uuidSuffix == "" {
		return "", errors.New("make id: uuidSuffix is empty")
	}
	return fmt.Sprintf(
		idFmt,
		idSchemeVersion,
		t.Unix(),
		(t.UnixNano() % (1000 * 1000 * 1000)),
		uuidSuffix,
	), nil
}
