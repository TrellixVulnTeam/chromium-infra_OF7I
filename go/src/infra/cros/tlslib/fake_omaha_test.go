// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// +build !windows

package tlslib

import (
	"bytes"
	"context"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"go.chromium.org/chromiumos/config/go/api/test/tls"
	"google.golang.org/grpc"

	"infra/cros/tlslib/internal/nebraska"
)

// Flags needed for integration tests which depend on real DUTs and networking.
var (
	wiringAddr = flag.String("wiring-addr", "127.0.0.1:7152", "address (host:port) to use for tlslib integration tests")
	dutName    = flag.String("dut", "", "DUT name to use for tlslib integration tests")
	build      = flag.String("build", "banjo-release/R90-13809.0.0", `build (in format of "<board>-<build_type>/<version>") to use for tlslib integration tests`)
)

// TestFakeOmahaIntegration tests CreateFakeOmaha and DeleteFakeOmaha in a real
// environment.
// The requirements are:
//     1) a workable TLW implementation;
//     2) locally installed `gsutil` and authorized to
//        "gs://chromeos-image-archive";
//     3) a SSH-able DUT.
// Command line to run:
//     go test -dut <DUT_name> ./...
// or you can use "-run TestFakeOmahaIntegration" instead of "./..." to just run
// this test.
func TestFakeOmahaIntegration(t *testing.T) {
	t.Parallel()
	if *dutName == "" {
		t.Skip("Skipping because no DUT specified")
	}
	if info, err := os.Stat(nebraska.Script); os.IsNotExist(err) || info.IsDir() {
		t.Skipf("Skipping because nebraska script %q doesn't exist or isn't a file", nebraska.Script)
	}
	connTlw, err := grpc.Dial(*wiringAddr, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Connect to %q failed (forgot to start the TLW?)", *wiringAddr)
	}
	t.Cleanup(func() { connTlw.Close() })
	s, err := NewServer(timeoutCtx(t, 2*time.Second), connTlw)
	if err != nil {
		t.Fatalf("NewServer: %s", err)
	}
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("NewServer: %s", err)
	}
	t.Cleanup(func() { l.Close() })
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.Serve(l)
	}()
	t.Cleanup(func() {
		s.GracefulStop()
		wg.Wait()
	})

	conn, err := grpc.Dial(l.Addr().String(), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("connect to TLS server: %s", err)
	}
	t.Cleanup(func() { conn.Close() })
	c := tls.NewCommonClient(conn)

	var rsp *tls.FakeOmaha
	t.Run("CreateFakeOmaha", func(t *testing.T) {
		rsp, err = c.CreateFakeOmaha(timeoutCtx(t, 20*time.Second), &tls.CreateFakeOmahaRequest{
			FakeOmaha: &tls.FakeOmaha{
				Dut: *dutName,
				TargetBuild: &tls.ChromeOsImage{
					PathOneof: &tls.ChromeOsImage_GsPathPrefix{
						// It doesn't matter whether the board in below URL match
						// with the DUT board.
						GsPathPrefix: fmt.Sprintf("gs://chromeos-image-archive/%s", *build),
					},
				},
				Payloads: []*tls.FakeOmaha_Payload{{Type: tls.FakeOmaha_Payload_FULL}},
			},
		})
		if err != nil {
			t.Fatalf("CreateFakeOmaha() error: %s", err)
		}

		if prefix := "fakeOmaha/"; !strings.HasPrefix(rsp.Name, prefix) {
			t.Errorf("CreateFakeOmaha() error: resource name %q not start with %q", rsp.Name, prefix)
		}
		t.Logf("The Omaha URL is %q", rsp.OmahaUrl)

		const fakeAURequest = `<?xml version="1.0" encoding="UTF-8"?>
<request requestid="1bcea19b-8ecf-4599-b37a-47018b7b8ecb" sessionid="710055d0-f9ec-4efd-aa8b-3ab153e4e0e9" protocol="3.0" updater="ChromeOSUpdateEngine" updaterversion="0.1.0.0" installsource="ondemandupdate" ismachine="1">
    <os version="Indy" platform="Chrome OS" sp="13336.0.0_x86_64"></os>
    <app appid="{3A837630-D749-4B7A-86C1-DB0ECC07A08B}" version="13336.0.0" track="stable-channel" board="banjo" hardware_class="BANJO C7A-C6I-A4O" delta_okay="true" installdate="4935" lang="en-US" fw_version="" ec_version="" >
        <updatecheck></updatecheck>
    </app>
</request>
`
		stream, err := c.ExecDutCommand(timeoutCtx(t, 2*time.Second), &tls.ExecDutCommandRequest{
			Name:    *dutName,
			Command: "curl",
			Args:    []string{"-X", "POST", "-d", "@-", "-H", "content-type:application/xml", rsp.OmahaUrl},
			Stdin:   []byte(fakeAURequest),
		})
		if err != nil {
			t.Fatalf("exec dut command error: %s", err)
		}
		var stdout bytes.Buffer
	readStream:
		for {
			rsp, err := stream.Recv()
			switch err {
			case nil:
				stdout.Write(rsp.Stdout)
			case io.EOF:
				break readStream
			default:
				t.Fatalf("ExecDutCommand RPC error: %s", err)
			}
		}
		// We think the test is good as long as receiving a valid xml response.
		if err := xml.Unmarshal(stdout.Bytes(), new(interface{})); err != nil {
			t.Errorf("TestFakeOmahaIntegration: failed to unmarshal response: %s. Want a valid xml, got %q", err, stdout.String())
		} else {
			t.Logf("receive output: %s", stdout.String())
		}
	})
	t.Run("DeleteFakeOmaha", func(t *testing.T) {
		_, err = s.DeleteFakeOmaha(timeoutCtx(t, 2*time.Second), &tls.DeleteFakeOmahaRequest{Name: rsp.GetName()})
		if err != nil {
			t.Errorf("DeleteFakeOmaha(%q) error: %q", rsp.GetName(), err)
		}
	})
}

func TestCreateFakeOmahaErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		req  *tls.CreateFakeOmahaRequest
	}{
		{"nil", nil},
		{
			"just dut name",
			&tls.CreateFakeOmahaRequest{
				FakeOmaha: &tls.FakeOmaha{Dut: "dutname"},
			},
		},
		{
			"no payload type",
			&tls.CreateFakeOmahaRequest{
				FakeOmaha: &tls.FakeOmaha{
					Dut: "dutname",
					TargetBuild: &tls.ChromeOsImage{
						PathOneof: &tls.ChromeOsImage_GsPathPrefix{
							GsPathPrefix: "gs://chromeos-image-archive/eve-release/R90-13809.0.0",
						},
					},
				},
			},
		},
	}
	s, err := NewServer(timeoutCtx(t, time.Second), nil)
	if err != nil {
		t.Fatalf("New TLS server: %s", err)
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := s.CreateFakeOmaha(timeoutCtx(t, time.Second), tc.req)
			if err == nil {
				t.Errorf("CreateFakeOmaha(%q) succeeded without all required arguments , want error", tc.req)
			}
		})
	}
}

func timeoutCtx(t *testing.T, d time.Duration) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	t.Cleanup(cancel)
	return ctx
}
