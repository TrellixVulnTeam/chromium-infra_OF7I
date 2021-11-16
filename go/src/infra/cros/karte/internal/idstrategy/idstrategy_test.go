// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package idstrategy

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

// TestMakeRawID tests that MakeRawID makes an ID.
func TestMakeRawID(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		in     time.Time
		suffix uint32
		out    string
	}{
		{
			name:   "good ID",
			in:     time.Unix(1, 2),
			suffix: 1,
			out:    "zzzzUzzzzzzzzzs0000000000F",
		},
		{
			name:   "no suffix",
			in:     time.Unix(1, 2),
			suffix: 0,
			out:    "zzzzUzzzzzzzzzs00000000000",
		},
		{
			name:   "no suffix",
			in:     time.Unix(2, 2),
			suffix: 0,
			out:    "zzzzUzzzzzzzzzo00000000000",
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			id, err := makeRawID(tt.in, tt.suffix)
			if err != nil {
				t.Errorf("unexpected error: %s", err)
			}
			if diff := cmp.Diff(id, tt.out); diff != "" {
				t.Errorf("(-want +got): %s", diff)
			}
		})
	}
}

// TestForgingKeyWithCorrectTypeShouldSucceed tests that we can replace the current strategy if we can see the "key" type, which is private.
func TestForgingKeyWithCorrectTypeShouldSucceed(t *testing.T) {
	t.Parallel()
	oldStrategy := NewNaive()
	newStrategy := NewNaive()
	if oldStrategy == newStrategy {
		panic("old and new strategies must have different addresses!")
	}

	ctx := context.Background()
	ctx = Use(ctx, oldStrategy)
	forgedKey := key("strategy key")
	ctx = context.WithValue(ctx, forgedKey, newStrategy)

	expected := newStrategy
	actual := Get(ctx)

	if expected == actual {
		// Do nothing. Test Successful.
	} else {
		t.Errorf("replacement was unexpectedly unsuccessful")
	}
}

// TestForgingKeyWithWrongTypeShouldFail tests that we CANNOT replace the strategy by using an ordinary string that happens to have the correct value.
func TestForgingKeyWithWrongTypeShouldFail(t *testing.T) {
	t.Parallel()
	oldStrategy := NewNaive()
	newStrategy := NewNaive()
	if oldStrategy == newStrategy {
		panic("old and new strategies must have different addresses!")
	}

	ctx := context.Background()
	ctx = Use(ctx, oldStrategy)
	forgedKey := "strategy key"
	ctx = context.WithValue(ctx, forgedKey, newStrategy)

	expected := oldStrategy
	actual := Get(ctx)

	if expected == actual {
		// Do nothing. Test successful.
	} else {
		t.Errorf("unexpectedly replaced strategy with bad key!")
	}
}
