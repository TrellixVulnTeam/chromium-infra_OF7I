// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tlsutil

import (
	"context"
	"net"

	"go.chromium.org/chromiumos/infra/proto/go/tls"
	"golang.org/x/crypto/ssh"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

// MakeTestClient returns a gRPC client connected to a listener which
// is implemented using an in-memory buffer.
// This is to be used for testing.
func MakeTestClient(ctx context.Context) (*grpc.ClientConn, net.Listener) {
	l := bufconn.Listen(1 << 10)
	dialer := func(ctx context.Context, address string) (net.Conn, error) {
		return l.Dial()
	}
	c, err := grpc.DialContext(ctx, "ignored", grpc.WithContextDialer(dialer), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	return c, l
}

// WiringFake is a fake implementation of tls.WiringServer for testing.
type WiringFake struct {
	tls.UnimplementedWiringServer
	DUTAddress string
}

// Serve serves the service using the listener.
func (s WiringFake) Serve(l net.Listener) error {
	server := grpc.NewServer()
	tls.RegisterWiringServer(server, &s)
	return server.Serve(l)
}

// OpenDutPort implements the respective gRPC.
func (s WiringFake) OpenDutPort(ctx context.Context, req *tls.OpenDutPortRequest) (*tls.OpenDutPortResponse, error) {
	return &tls.OpenDutPortResponse{
		Status:  tls.OpenDutPortResponse_STATUS_OK,
		Address: s.DUTAddress,
	}, nil
}

// SSHStub is a stub implementation of an SSH server for testing.
type SSHStub struct {
	Output     []byte
	ExitStatus uint32
	Logger     Logger
}

// Serve serves the service using the listener.
func (s SSHStub) Serve(l net.Listener) error {
	for {
		c, err := l.Accept()
		if err != nil {
			return err
		}
		go s.handleFakeSSHConn(c)
	}
}

const (
	testServerKey = `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACCcxPrNI3zLCQvi06KulSkiTn9hYUvsOXmO5+nhGDVRSAAAAJiJwd+hicHf
oQAAAAtzc2gtZWQyNTUxOQAAACCcxPrNI3zLCQvi06KulSkiTn9hYUvsOXmO5+nhGDVRSA
AAAEAnXgWcZ2ZxrpciL0TjP8yaEwvTm1wEd0md1F0X7y6It5zE+s0jfMsJC+LToq6VKSJO
f2FhS+w5eY7n6eEYNVFIAAAAD2F5YXRhbmVAcmFjaWVsYQECAwQFBg==
-----END OPENSSH PRIVATE KEY-----
`
)

func (s SSHStub) handleFakeSSHConn(c net.Conn) {
	config := &ssh.ServerConfig{
		NoClientAuth: true,
	}
	key, err := ssh.ParsePrivateKey([]byte(testServerKey))
	if err != nil {
		panic(err)
	}
	config.AddHostKey(key)

	_, chans, reqs, err := ssh.NewServerConn(c, config)
	if err != nil {
		s.Logger.Printf("SSHStub: %s", err)
		return
	}
	go ssh.DiscardRequests(reqs)
	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}
		channel, requests, err := newChannel.Accept()
		if err != nil {
			s.Logger.Printf("SSHStub: %s", err)
			return
		}

		block := make(chan struct{}, 1)
		go func(in <-chan *ssh.Request) {
			for req := range in {
				s.Logger.Printf("SSHStub: got request: %#v", req)
				req.Reply(req.Type == "exec", nil)
				if req.Type == "exec" {
					block <- struct{}{}
				}
			}
		}(requests)

		go func() {
			defer channel.Close()
			<-block
			_, err = channel.Write(s.Output)
			if err != nil {
				s.Logger.Printf("SSHStub: write output: %s", err)
			}
			_, _ = channel.SendRequest("exit-status", false, ssh.Marshal(exitStatusMsg{s.ExitStatus}))
		}()
	}
}

type exitStatusMsg struct {
	exitStatus uint32
}

// Logger is the interface used for a logging sink.
type Logger interface {
	Printf(string, ...interface{})
}
