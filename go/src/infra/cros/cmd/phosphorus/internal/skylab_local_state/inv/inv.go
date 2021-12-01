// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package inv provides option to build connection to Inventory server service.
package inv

import (
	"go.chromium.org/luci/common/errors"
	"google.golang.org/grpc"

	labapi "go.chromium.org/chromiumos/config/go/test/lab/api"
)

const (
	inventoryServicePort = ":1485"
)

// InventoryService represents an InventoryServiceClient and the connection it uses
type InventoryService struct {
	Client labapi.InventoryServiceClient
	conn   *grpc.ClientConn
}

// NewClient initialize and return new client to work with Inventory server service.
func NewClient() (*InventoryService, error) {
	conn, err := grpc.Dial(inventoryServicePort, grpc.WithInsecure())
	if err != nil {
		return nil, errors.Annotate(err, "Dial").Err()
	}

	cl := labapi.NewInventoryServiceClient(conn)

	return &InventoryService{
		Client: cl,
		conn:   conn,
	}, nil
}

// Close client cleans up resources associated with InventoryService
func CloseClient(invServ *InventoryService) error {
	// Make it safe to closeClient() more than once
	if invServ.Client == nil {
		return nil
	}
	err := invServ.conn.Close()
	invServ.Client = nil
	return err
}
