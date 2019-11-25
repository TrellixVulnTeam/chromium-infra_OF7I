// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Command thin-tls is a thin/fake implementation of the TLS API.
// This is not finalized yet (don't depend on backward compatibility).
// See README for more info.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"

	"google.golang.org/grpc"

	"infra/cmd/cros/thin-tls/api"
)

var (
	address    string
	configPath string
)

func init() {
	flag.StringVar(&address, "address", ":50051", "Service listen address")
	flag.StringVar(&configPath, "config", "thin-tls-config.json", "JSON config file path")
}

func main() {
	flag.Parse()

	c, err := loadConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}

	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	api.RegisterTlsServer(s, &server{
		config: c,
	})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

type config struct {
	DutHostname string `json:"dutHostname"`
}

func loadConfig(path string) (*config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	d, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	var c config
	if err := json.Unmarshal(d, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

type server struct {
	api.UnimplementedTlsServer
	config *config
}

func (s *server) DutShell(req *api.DutShellRequest, stream api.Tls_DutShellServer) error {
	cmd := exec.Command("ssh", s.config.DutHostname, req.GetCommand())
	outw := bufio.NewWriter(dutShellStdoutWriter{stream})
	errw := bufio.NewWriter(dutShellStderrWriter{stream})
	cmd.Stdout = outw
	cmd.Stderr = errw
	status := int32(0)
	if err := cmd.Run(); err != nil {
		if err, ok := err.(*exec.ExitError); ok {
			status = int32(err.ExitCode())
		} else {
			_ = outw.Flush()
			_ = errw.Flush()
			return err
		}
	}
	if err := outw.Flush(); err != nil {
		return err
	}
	if err := errw.Flush(); err != nil {
		return err
	}
	return stream.Send(&api.DutShellResponse{
		Status: status,
		Exited: true,
	})
}

// dutShellStdoutWriter wraps a DutShell stream as an io.Writer.
type dutShellStdoutWriter struct {
	stream api.Tls_DutShellServer
}

func (w dutShellStdoutWriter) Write(p []byte) (n int, err error) {
	resp := api.DutShellResponse{
		Stdout: p,
	}
	if err := w.stream.Send(&resp); err != nil {
		return 0, fmt.Errorf("dut shell writer: %v", err)
	}
	return len(p), nil
}

// dutShellStderrWriter wraps a DutShell stream as an io.Writer.
type dutShellStderrWriter struct {
	stream api.Tls_DutShellServer
}

func (w dutShellStderrWriter) Write(p []byte) (n int, err error) {
	resp := api.DutShellResponse{
		Stderr: p,
	}
	if err := w.stream.Send(&resp); err != nil {
		return 0, fmt.Errorf("dut shell writer: %v", err)
	}
	return len(p), nil
}
