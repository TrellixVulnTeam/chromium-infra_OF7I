// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Command thin-tls is a thin/fake implementation of the TLS API.
// This is not finalized yet (don't depend on backward compatibility).
// See README for more info.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net"
	"os"

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

func (s *server) AvailableServices(ctx context.Context, req *api.AvailableServicesRequest) (*api.AvailableServicesResponse, error) {
	return &api.AvailableServicesResponse{}, nil
}
