// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/uuid"
	"go.chromium.org/chromiumos/config/go/api/test/tls/dependencies/longrunning"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// operation is used by lroManager to hold extra metadata.
type operation struct {
	op           *longrunning.Operation
	creationTime time.Time
}

type lroManager struct {
	mu sync.Mutex
	// Provide stubs for unimplemented methods
	longrunning.UnimplementedOperationsServer
	// Mapping of Operation name to Operation.
	operations map[string]*operation
	// expiryStopper signals the expiration goroutine to terminate.
	expiryStopper chan struct{}
}

// newLROManager returns a new lroManager which must be closed after use.
func newLROManager() *lroManager {
	m := &lroManager{
		operations:    make(map[string]*operation),
		expiryStopper: make(chan struct{}),
	}
	go func() {
		for {
			select {
			case <-m.expiryStopper:
				return
			case <-time.After(time.Hour):
				m.deleteExpiredOperations()
			}
		}
	}()
	return m
}

func (m *lroManager) Close() {
	close(m.expiryStopper)
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
		m.operations[name] = &operation{
			op: &longrunning.Operation{
				Name: name,
			},
			creationTime: time.Now(),
		}
		return proto.Clone(m.operations[name].op).(*longrunning.Operation)
	}
}

func (m *lroManager) delete(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.operations[name]; !ok {
		return fmt.Errorf("lroManager delete: unkown name %s", name)
	}
	delete(m.operations, name)
	return nil
}

func (m *lroManager) deleteExpiredOperations() {
	m.mu.Lock()
	defer m.mu.Unlock()
	cutoff := time.Now().Add(-24 * time.Hour)
	for name, op := range m.operations {
		if op.creationTime.Before(cutoff) {
			delete(m.operations, name)
		}
	}
}

func (m *lroManager) update(newOp *longrunning.Operation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	name := newOp.Name
	if _, ok := m.operations[name]; !ok {
		return fmt.Errorf("lroManager update: unknown name %s", name)
	}
	m.operations[name].op = proto.Clone(newOp).(*longrunning.Operation)
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
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("GetOperation: Operation name %s does not exist", name))
	}
	return v.op, nil
}

func (m *lroManager) DeleteOperation(ctx context.Context, in *longrunning.DeleteOperationRequest) (*empty.Empty, error) {
	name := in.Name
	if err := m.delete(name); err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("DeleteOperation: failed to delete Operation name %s, %s", name, err))
	}
	return &empty.Empty{}, nil
}

func (m *lroManager) CancelOperation(ctx context.Context, in *longrunning.CancelOperationRequest) (*empty.Empty, error) {
	return &empty.Empty{}, status.Error(codes.Unimplemented, "CancelOperation is not implemented")
}
