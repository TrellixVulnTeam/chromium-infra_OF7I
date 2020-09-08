// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/uuid"
	"go.chromium.org/chromiumos/config/go/api/test/tls/dependencies/longrunning"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type lroManager struct {
	mu sync.Mutex
	// Mapping of Operation name to Operation.
	operations map[string]*longrunning.Operation
}

func newLROManager() *lroManager {
	return &lroManager{
		operations: make(map[string]*longrunning.Operation),
	}
}

func (m *lroManager) new() *longrunning.Operation {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Loop few times until a unique UUID is created.
	for iterations := 0; ; iterations++ {
		if iterations >= 5 {
			panic("Could not find a unique UUID for a new Operation")
		}
		name := uuid.New().String()
		if _, ok := m.operations[name]; ok {
			continue
		}
		op := &longrunning.Operation{
			Name: name,
		}
		m.operations[name] = op
		return proto.Clone(op).(*longrunning.Operation)
	}
}

func (m *lroManager) update(newOp *longrunning.Operation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	name := newOp.Name
	if _, ok := m.operations[name]; !ok {
		return fmt.Errorf("lroManager update: unknown name %s", name)
	}
	m.operations[name] = proto.Clone(newOp).(*longrunning.Operation)
	return nil
}

func (m *lroManager) ListOperations(ctx context.Context, in *longrunning.ListOperationsRequest) (*longrunning.ListOperationsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Listoperations is not implemented")
}

func (m *lroManager) GetOperation(ctx context.Context, in *longrunning.GetOperationRequest) (*longrunning.Operation, error) {
	name := in.Name
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.operations[name]
	if !ok {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("GetOperation: Operation name %#v does not exist", name))
	}
	return v, nil
}

func (m *lroManager) DeleteOperation(ctx context.Context, in *longrunning.DeleteOperationRequest) (*empty.Empty, error) {
	name := in.Name
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.operations[name]; !ok {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("DeleteOperation: Operation name %#v does not exist", name))
	}
	delete(m.operations, name)
	return &empty.Empty{}, nil
}

func (m *lroManager) CancelOperation(ctx context.Context, in *longrunning.CancelOperationRequest) (*empty.Empty, error) {
	return &empty.Empty{}, status.Error(codes.Unimplemented, "CancelOperation is not implemented")
}
