// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package shared

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDoWithRetry_error(t *testing.T) {
	retries := 5
	opts := Options{
		BackoffBase: 0,
		BaseDelay:   0 * time.Second,
		Retries:     retries,
	}
	count := 0
	err := DoWithRetry(context.Background(), opts, func() error {
		count++
		return errors.New("throw")
	})
	// assert retries + 1 attempts: 1 failed attempt + n retries
	if count != retries+1 {
		t.Errorf("Expected %d attempts, but got %d", retries+1, count)
	}
	if err == nil {
		t.Errorf("Expected an error, but got none")
	}
}

func TestDoWithRetry_success(t *testing.T) {
	retries := 5
	opts := Options{
		BackoffBase: 0,
		BaseDelay:   0 * time.Second,
		Retries:     retries,
	}
	count := 0
	err := DoWithRetry(context.Background(), opts, func() error {
		count++
		return nil
	})
	if count != 1 {
		t.Errorf("Expected 1 attempt, but got %d", count)
	}
	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}
}
