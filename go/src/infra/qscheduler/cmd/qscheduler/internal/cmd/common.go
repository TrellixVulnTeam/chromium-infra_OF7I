// Copyright 2018 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/grpc/prpc"

	"infra/qscheduler/cmd/qscheduler/internal/site"
	qscheduler "infra/qscheduler/service/api/qscheduler/v1"
)

const progName = "qscheduler"

type envFlags struct {
	dev bool
}

func (f *envFlags) Register(fl *flag.FlagSet) {
	fl.BoolVar(&f.dev, "dev", false, "Run in dev environment")
}

func (f envFlags) Env() site.Environment {
	if f.dev {
		return site.Dev
	}
	return site.Prod
}

// httpClient returns an HTTP client with authentication set up.
func httpClient(ctx context.Context, f *authcli.Flags) (*http.Client, error) {
	o, err := f.Options()
	if err != nil {
		return nil, errors.Annotate(err, "failed to get auth options").Err()
	}
	a := auth.NewAuthenticator(ctx, auth.OptionalLogin, o)
	c, err := a.Client()
	if err != nil {
		return nil, errors.Annotate(err, "failed to create HTTP client").Err()
	}
	return c, nil
}

func prpcClient(ctx context.Context, a *authcli.Flags, e *envFlags) (*prpc.Client, error) {
	h, err := httpClient(ctx, a)
	if err != nil {
		return nil, err
	}

	return &prpc.Client{
		C:       h,
		Host:    e.Env().QSchedulerHost,
		Options: site.DefaultPRPCOptions,
	}, nil
}

func newAdminClient(ctx context.Context, a *authcli.Flags, e *envFlags) (qscheduler.QSchedulerAdminClient, error) {
	p, err := prpcClient(ctx, a, e)
	if err != nil {
		return nil, err
	}

	return qscheduler.NewQSchedulerAdminPRPCClient(p), nil
}

func newViewClient(ctx context.Context, a *authcli.Flags, e *envFlags) (qscheduler.QSchedulerViewClient, error) {
	p, err := prpcClient(ctx, a, e)
	if err != nil {
		return nil, err
	}

	return qscheduler.NewQSchedulerViewPRPCClient(p), nil
}

func toFloats(s []string) ([]float32, error) {
	floats := make([]float32, len(s))
	for i, c := range s {
		f, err := strconv.ParseFloat(c, 32)
		if err != nil {
			return nil, err
		}
		floats[i] = float32(f)
	}
	return floats, nil
}

// int32DimsVar implements the flag.Value interface. It provides a keyval flag
// handler for `add-account` and `mod-account` to parse comma-separated keyval
// flags into a map[string]int32.
type int32DimsVar struct {
	handle *map[string]int32
}

// String returns the default value for dimensions represented as a string. The
// default value is an empty map, which stringifies to an empty string.
func (int32DimsVar) String() string {
	return ""
}

// Set populates the dims map with comma-separated keyval pairs.
func (d int32DimsVar) Set(keyvals string) error {
	if d.handle == nil {
		panic("int32DimsVar handle must be pointing to a map[string]string!")
	}
	if *d.handle == nil {
		*d.handle = map[string]int32{}
	}
	// strings.Split, if given an empty string, will produce a
	// slice containing a single string.
	if keyvals == "" {
		return nil
	}
	m := *d.handle
	for _, entry := range strings.Split(keyvals, ",") {
		key, val, err := splitKeyVal(entry)
		if err != nil {
			return err
		}
		if _, exists := m[key]; exists {
			return fmt.Errorf("key %q is already specified", key)
		}
		m[key] = val
	}
	return nil
}

// int32KeyVals takes an initial map and produces a flag variable that can be
// set from command line arguments
func int32KeyVals(m *map[string]int32) flag.Value {
	if m == nil {
		panic("Argument to int32KeyVals must be pointing to a map[string]int32!")
	}
	return int32DimsVar{handle: m}
}

// splitKeyVal splits a string with "=" or ":" into a string-int32 key-value
// pair, and returns an error if this is impossible. Strings with multiple "="
// or ":" values are considered malformed.
func splitKeyVal(s string) (string, int32, error) {
	re := regexp.MustCompile("[=:]")
	res := re.Split(s, -1)
	if len(res) != 2 {
		return "", 0, fmt.Errorf(`string %q is a malformed key-value pair`, s)
	}
	intVal, err := strconv.ParseInt(res[1], 10, 32)
	if err != nil {
		return "", 0, err
	}
	return res[0], int32(intVal), nil
}
