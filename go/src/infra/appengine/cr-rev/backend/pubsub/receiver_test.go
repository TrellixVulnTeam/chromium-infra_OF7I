// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package pubsub

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"unsafe"

	"cloud.google.com/go/pubsub"
	. "github.com/smartystreets/goconvey/convey"
)

// ackHandler interface matches pubsub/message ackHandler
type ackHandler interface {
	OnAck()
	OnNack()
}

//
type fakeAckHandler struct {
	acked  int
	nacked int
	mu     sync.Mutex
}

func (ackh *fakeAckHandler) OnAck() {
	ackh.mu.Lock()
	defer ackh.mu.Unlock()
	ackh.acked++
}

func (ackh *fakeAckHandler) OnNack() {
	ackh.mu.Lock()
	defer ackh.mu.Unlock()
	ackh.nacked++
}

type mockPubsubReceiver struct {
	messages   []*pubsub.Message
	ackHandler ackHandler
}

func (m *mockPubsubReceiver) Receive(ctx context.Context, f func(ctx context.Context, m *pubsub.Message)) error {
	for _, message := range m.messages {
		// Replace ackHandler with fake handler.
		pointer := reflect.ValueOf(message)
		val := reflect.Indirect(pointer)
		mem := val.FieldByName("ackh")
		ptrToMem := unsafe.Pointer(mem.UnsafeAddr())
		realPtrToMem := (*ackHandler)(ptrToMem)
		*realPtrToMem = m.ackHandler
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
		ackh := &fakeAckHandler{}
		mReceiver := &mockPubsubReceiver{
			messages:   make([]*pubsub.Message, 0),
			ackHandler: ackh,
		}
		mProcess := &mockProcessMessage{}

		err := Subscribe(ctx, mReceiver, mProcess.processPubsubMessage)
		So(err, ShouldBeNil)
		So(ackh.acked, ShouldEqual, 0)
		So(ackh.nacked, ShouldEqual, 0)
	})

	Convey("invalid message", t, func() {
		ctx := context.Background()
		ackh := &fakeAckHandler{}
		mReceiver := &mockPubsubReceiver{
			messages: []*pubsub.Message{
				{
					Data: []byte("foo"),
				},
			},
			ackHandler: ackh,
		}
		mProcess := &mockProcessMessage{}

		err := Subscribe(ctx, mReceiver, mProcess.processPubsubMessage)
		So(err, ShouldBeNil)
		So(ackh.acked, ShouldEqual, 0)
		So(ackh.nacked, ShouldEqual, 1)
	})

	Convey("valid message", t, func() {
		ctx := context.Background()
		ackh := &fakeAckHandler{}
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
			ackHandler: ackh,
		}
		mProcess := &mockProcessMessage{}

		err := Subscribe(ctx, mReceiver, mProcess.processPubsubMessage)
		So(err, ShouldBeNil)
		So(ackh.acked, ShouldEqual, 1)
		So(ackh.nacked, ShouldEqual, 0)
		So(mProcess.calls, ShouldEqual, 1)
	})
}
