// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"net"
	"testing"

	"infra/cros/tlsutil"

	"github.com/golang/protobuf/proto"
	"github.com/google/go-cmp/cmp"
	"go.chromium.org/chromiumos/infra/proto/go/tls"
)

func TestDutShell(t *testing.T) {
	t.Parallel()
	sshListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer sshListener.Close()
	sshStub := tlsutil.SSHStub{
		Output:     []byte("ayanami"),
		ExitStatus: 15,
		Logger:     testLogger{t},
	}
	go sshStub.Serve(sshListener)

	ctx := context.Background()
	wiringConn, wiringListener := tlsutil.MakeTestClient(ctx)
	defer wiringListener.Close()
	wiringFake := tlsutil.WiringFake{
		DUTAddress: sshListener.Addr().String(),
	}
	go wiringFake.Serve(wiringListener)

	serverConn, serverListener := tlsutil.MakeTestClient(ctx)
	defer serverListener.Close()
	s := server{
		conn: wiringConn,
	}
	go s.Serve(serverListener)

	c := tls.NewCommonClient(serverConn)
	stream, err := c.DutShell(ctx, &tls.DutShellRequest{
		Dut:     "some-dut",
		Command: "echo hi",
	})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := stream.Recv()
	if err != nil {
		t.Fatal(err)
	}
	want := &tls.DutShellResponse{
		Status: 15,
		Exited: true,
		Stdout: []byte("ayanami"),
	}
	if !proto.Equal(resp, want) {
		diff := cmp.Diff(want, resp)
		t.Errorf("response differs: (-want +got)\n%s", diff)
	}
}

type testLogger struct {
	t *testing.T
}

func (t testLogger) Printf(format string, v ...interface{}) {
	t.t.Logf(format, v...)
}
