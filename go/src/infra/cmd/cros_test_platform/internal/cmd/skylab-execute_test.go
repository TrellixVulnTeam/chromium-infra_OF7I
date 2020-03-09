package cmd

import (
	"context"
	"testing"
	"time"

	"go.chromium.org/luci/common/errors"
)

func TestRunWithDeadlineSuccess(t *testing.T) {
	f := func(context.Context) error { return nil }
	terr, err := runWithDeadline(context.Background(), f, time.Now().Add(time.Hour))
	if terr != nil {
		t.Errorf("timeout error is not nil: %s", terr)
	}
	if err != nil {
		t.Errorf("error is not nil: %s", err)
	}
}

func TestRunWithDeadlineError(t *testing.T) {
	wantErr := errors.Reason("custom").Err()
	f := func(context.Context) error { return wantErr }
	terr, err := runWithDeadline(context.Background(), f, time.Now().Add(time.Hour))
	if terr != nil {
		t.Errorf("timeout error is not nil: %s", terr)
	}
	if err != wantErr {
		t.Errorf("incorrect error, want %s, got %s", wantErr, err)
	}
}

func TestRunWithDeadlineTimeoutError(t *testing.T) {
	f := func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	terr, err := runWithDeadline(context.Background(), f, time.Now().Add(-time.Hour))
	if terr == nil {
		t.Errorf("timeout error is nil despite past deadline")
	}
	if err != nil {
		t.Errorf("error is not nil: %s", err)
	}
}
