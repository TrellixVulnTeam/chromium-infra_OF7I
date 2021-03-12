// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package resource helps to manage AIP resources (https://google.aip.dev/121).
package resource

import (
	"fmt"
	"log"
	"sync"
)

// Resource represents a resource recommended by go/aip.
type Resource interface {
	// Close closes the resource.
	// It corresponds to any cleanup required by the standard AIP Delete method.
	// It is implementation dependent whether it is safe to call from multiple
	// goroutines.
	// The implementation must be safe to call multiple times.
	Close() error
}

// Manager tracks resources to support standard AIP methods like Get, Delete, as
// well as clean up runtime resources when the service exits.
type Manager struct {
	resources sync.Map
}

// NewManager returns a new instance of Manager.
func NewManager() *Manager {
	return &Manager{}
}

// Add adds a entry in the manager for a resource.
func (m *Manager) Add(name string, r Resource) error {
	if _, loaded := m.resources.LoadOrStore(name, r); loaded {
		return fmt.Errorf("existing resource name %q", name)
	}
	return nil
}

// Remove deletes a resource by name and return it to the caller.
func (m *Manager) Remove(name string) (Resource, error) {
	r, ok := m.resources.LoadAndDelete(name)
	if !ok {
		return nil, fmt.Errorf("unknown name %q", name)
	}
	return r.(Resource), nil
}

// Close closes the manager and all resources it tracks.
// Resources added and removed during the Close call may or may not be handled,
// and resources may be added and removed after calling close. The caller
// should ensure no more requests can happen before Close is called.
func (m *Manager) Close() {
	m.resources.Range(func(key, value interface{}) bool {
		if err := value.(Resource).Close(); err != nil {
			log.Printf("Resource manager: close %q error: %s", key.(string), err)
		}
		m.resources.Delete(key)
		return true
	})
}
