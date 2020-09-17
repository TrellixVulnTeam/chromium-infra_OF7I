// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package pubsub

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"
	"unsafe"

	"cloud.google.com/go/pubsub"
	. "github.com/smartystreets/goconvey/convey"
)

type mockPubsubReceiver struct {
	messages []*pubsub.Message
	calls    int
	acked    int
	mu       sync.Mutex
}

func (m *mockPubsubReceiver) Receive(ctx context.Context, f func(ctx context.Context, m *pubsub.Message)) error {
	for _, message := range m.messages {
		m.mu.Lock()
		m.calls++
		m.mu.Unlock()

		// Fix Ack method of message
		pointer := reflect.ValueOf(message)
		val := reflect.Indirect(pointer)
		mem := val.FieldByName("doneFunc")
		ptrToMem := unsafe.Pointer(mem.UnsafeAddr())
		realPtrToMem := (*func(string, bool, time.Time))(ptrToMem)
		*realPtrToMem = func(string, bool, time.Time) {
			m.mu.Lock()
			defer m.mu.Unlock()
			m.acked++
		}
		f(ctx, message)
	}
	return nil
}

type mockProcessMessage struct {
	calls int
}

func (m *mockProcessMessage) processPubsubMessage(ctx context.Context,
	event *SourceRepoEvent) error {
	m.calls++
	return nil
}

func TestPubsubSubscribe(t *testing.T) {
	Convey("no messages", t, func() {
		ctx := context.Background()
		mReceiver := &mockPubsubReceiver{
			messages: make([]*pubsub.Message, 0),
		}
		mProcess := &mockProcessMessage{}

		err := Subscribe(ctx, mReceiver, mProcess.processPubsubMessage)
		So(err, ShouldBeNil)
		So(mReceiver.calls, ShouldEqual, 0)
		So(mReceiver.acked, ShouldEqual, 0)
		So(mProcess.calls, ShouldEqual, 0)
	})

	Convey("invalid message", t, func() {
		ctx := context.Background()
		mReceiver := &mockPubsubReceiver{
			messages: []*pubsub.Message{
				{
					Data: []byte("foo"),
				},
			},
		}
		mProcess := &mockProcessMessage{}

		err := Subscribe(ctx, mReceiver, mProcess.processPubsubMessage)
		So(err, ShouldBeNil)
		So(mReceiver.calls, ShouldEqual, 1)
		So(mReceiver.acked, ShouldEqual, 0)
		So(mProcess.calls, ShouldEqual, 0)
	})

	Convey("valid message", t, func() {
		ctx := context.Background()
		mReceiver := &mockPubsubReceiver{
			messages: []*pubsub.Message{
				{
					Data: []byte(`
{
  "name": "projects/chromium-gerrit/repos/chromium/src",
  "url": "http://foo/",
  "eventTime": "2020-08-01T00:01:02.333333Z",
  "refUpdateEvent": {
    "refUpdates": {
      "refs/heads/master": {
        "refName": "refs/heads/master",
        "updateType": "UPDATE_FAST_FORWARD",
        "oldId": "b82e8bfe83fadac69a6cad56c06ec45b85c86e49",
        "newId": "ef279f3d5c617ebae8573a664775381fe0225e63"
      }
    }
  }
}`),
				},
			},
		}
		mProcess := &mockProcessMessage{}

		err := Subscribe(ctx, mReceiver, mProcess.processPubsubMessage)
		So(err, ShouldBeNil)
		So(mReceiver.calls, ShouldEqual, 1)
		So(mReceiver.acked, ShouldEqual, 1)
		So(mProcess.calls, ShouldEqual, 1)
	})
}
