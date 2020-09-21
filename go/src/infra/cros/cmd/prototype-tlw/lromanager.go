// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package main

import (
	"context"
	"fmt"
	"sync"
	"time"

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

func (m *lroManager) close() {
	close(m.expiryStopper)
}

func (m *lroManager) new() *longrunning.Operation {
	m.mu.Lock()
	defer m.mu.Unlock()
	name := uuid.New().String()
	if _, ok := m.operations[name]; ok {
		panic("Could not find a unique UUID for a new Operation")
	}
	m.operations[name] = &operation{
		op: &longrunning.Operation{
			Name: name,
		},
		creationTime: time.Now(),
	}
	return m.operations[name].op
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

func (m *lroManager) setResult(name string, result *longrunning.Operation_Response) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.operations[name]; !ok {
		return fmt.Errorf("lroManager setResult: unknown name %s", name)
	}
	m.operations[name].op.Done = true
	m.operations[name].op.Result = result
	return nil
}

func (m *lroManager) setError(name string, err *longrunning.Operation_Error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.operations[name]; !ok {
		return fmt.Errorf("lroManager setError: unknown name %s", name)
	}
	m.operations[name].op.Done = true
	m.operations[name].op.Result = err
	return nil
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
