// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// +build !windows

package nebraska

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"testing"
	"time"

	"go.chromium.org/chromiumos/config/go/api/test/tls"
)

type fakeEnv struct {
	startFakeNebraska func([]string) (Process, error)
}

func (e fakeEnv) DownloadMetadata(ctx context.Context, gsPathPrefix string, payloads []*tls.FakeOmaha_Payload, dir string) (string, error) {
	return "", nil
}

func (e fakeEnv) StartNebraska(cmdline []string) (Process, error) {
	return e.startFakeNebraska(cmdline)
}

var _ Environment = fakeEnv{}

type fakeProc struct{}

func (p fakeProc) Pid() int {
	return 0
}

func (p fakeProc) Stop() error {
	return nil
}

func (p fakeProc) Args() []string {
	return []string{"fake", "process"}
}

var _ Process = fakeProc{}

func TestNebraska_ParsePortFile(t *testing.T) {
	t.Parallel()
	const portWant = 12345
	port := []byte(strconv.Itoa(portWant))
	e := fakeEnv{
		startFakeNebraska: func(cmdline []string) (Process, error) {
			runtimeRoot := runtimeRootArg(cmdline)
			if runtimeRoot == "" {
				t.Fatalf("no -runtime-root specified in the command line: %#v", cmdline)
			}
			err := ioutil.WriteFile(path.Join(runtimeRoot, "port"), port, 0644)
			if err != nil {
				t.Fatalf("create fake port file: %s", err)
			}
			return &fakeProc{}, nil

		},
	}
	n, err := NewServer(context.Background(), e, "gs://", []*tls.FakeOmaha_Payload{}, "http://cache-server/update")
	if err != nil {
		t.Errorf("NewServer: failed to create Nebraska: %s", err)
	}
	t.Cleanup(func() {
		if err = n.Close(); err != nil {
			t.Errorf("close Nebraska: %s", err)
		}
	})
	if n.Port() != portWant {
		t.Errorf("NewServer: port: got %d, want %d", n.port, portWant)
	}
}

func TestNebraska_Close(t *testing.T) {
	t.Parallel()
	port := []byte("12345")
	e := fakeEnv{
		startFakeNebraska: func(cmdline []string) (Process, error) {
			runtimeRoot := runtimeRootArg(cmdline)
			if runtimeRoot == "" {
				t.Fatalf("no -runtime-root specified in the command line: %#v", cmdline)
			}
			err := ioutil.WriteFile(path.Join(runtimeRoot, "port"), port, 0644)
			if err != nil {
				t.Fatalf("create fake port file: %s", err)
			}
			return &fakeProc{}, nil
		},
	}
	n, err := NewServer(context.Background(), e, "gs://", []*tls.FakeOmaha_Payload{}, "http://cache-server/update")
	if err != nil {
		t.Errorf("NewServer: failed to create Nebraska: %s", err)
	}

	if err = n.Close(); err != nil {
		t.Errorf("close Nebraska: %s", err)
	}
	if _, err := os.Stat(n.runtimeRoot); err == nil {
		t.Errorf("close Nebraska: runtime root was not removed")
	}
	if n.proc != nil {
		t.Errorf("close Nebraska: process was not terminated")
	}
}

func TestNebraska_TimeoutOnPort(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping test in short mode.")
	}
	e := fakeEnv{
		startFakeNebraska: func([]string) (Process, error) {
			return &fakeProc{}, nil
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := NewServer(ctx, e, "gs://", []*tls.FakeOmaha_Payload{}, "http://cache-server/update")
	if err == nil {
		t.Fatalf("NewServer() succeeded without Nebraska port file, want error")
	}
}

func TestEnv_DownloadMetadata(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		types    []*tls.FakeOmaha_Payload
		patterns []string
	}{
		{
			"full payload only",
			[]*tls.FakeOmaha_Payload{{Type: tls.FakeOmaha_Payload_FULL}},
			[]string{fullPayloadPattern},
		},
		{
			"delta payload only",
			[]*tls.FakeOmaha_Payload{{Type: tls.FakeOmaha_Payload_DELTA}},
			[]string{deltaPayloadPattern},
		},
		{
			"full and delta payload",
			[]*tls.FakeOmaha_Payload{{Type: tls.FakeOmaha_Payload_FULL}, {Type: tls.FakeOmaha_Payload_DELTA}},
			[]string{fullPayloadPattern, deltaPayloadPattern},
		},
	}
	gsPrefix := "gs://bucket/build/version"
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := ""
			e := env{
				runCmd: func(ctx context.Context, name string, args ...string) *exec.Cmd {
					got = fmt.Sprintf("%s %s", name, strings.Join(args, " "))
					return exec.Command("true")
				},
			}
			e.DownloadMetadata(context.Background(), gsPrefix, tc.types, "")
			w := []string{"gsutil cp"}
			for _, p := range tc.patterns {
				w = append(w, fmt.Sprintf("%s/%s", gsPrefix, p))
			}
			prefix := strings.Join(w, " ")
			if !strings.HasPrefix(got, prefix) {
				t.Errorf("DownloadMetadata(FULL) error: want prefix %q, got %q", prefix, got)
			}
		})
	}
}

func runtimeRootArg(cmdline []string) string {
	for i, arg := range cmdline {
		if arg == "--runtime-root" && i < len(cmdline)-1 {
			return cmdline[i+1]
		}
	}
	return ""
}
