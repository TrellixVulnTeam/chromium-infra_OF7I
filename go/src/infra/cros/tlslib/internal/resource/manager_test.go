// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package resource

import (
	"testing"
)

type fakeResource struct {
	closed bool
}

func (f *fakeResource) Close() error {
	f.closed = true
	return nil
}

func TestResourceManager(t *testing.T) {
	t.Parallel()
	m := NewManager()
	var r fakeResource
	t.Run("Add a new resource", func(t *testing.T) {
		const name = "name"
		err := m.Add(name, &r)
		if err != nil {
			t.Errorf("Add(%q, &%#v) failed: %s", name, r, err)
		}
	})
	t.Run("Add resource with conflict name", func(t *testing.T) {
		const name = "name"
		err := m.Add(name, &r)
		if err == nil {
			t.Errorf("Add(%q, &%#v) succeeded for duplicate name, want error", name, r)
		}
	})
	t.Run("Remove an existing resource", func(t *testing.T) {
		const name = "name"
		want := r
		got, err := m.Remove(name)
		if err != nil {
			t.Errorf(`Remove(%q) failed: %s`, name, err)
		}
		if *(got.(*fakeResource)) != want {
			t.Errorf("Remove(%q) = %#v, want %#v", name, got, want)
		}
	})
	t.Run("Remove non existing resource", func(t *testing.T) {
		const name = "name"
		_, err := m.Remove(name)
		if err == nil {
			t.Errorf("Remove(%q) succeeded for non existing resource, want error", name)
		}
	})
}

func TestResourceManager_Close(t *testing.T) {
	t.Parallel()
	m := NewManager()
	var r fakeResource
	const name = "name"
	if err := m.Add(name, &r); err != nil {
		t.Errorf("Add(%q, &%#v) failed: %s", name, r, err)
	}
	m.Close()
	if !r.closed {
		t.Errorf("Close() didn't close the added resource %q", name)
	}
}
