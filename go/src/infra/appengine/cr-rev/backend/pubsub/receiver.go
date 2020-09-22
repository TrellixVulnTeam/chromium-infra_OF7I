// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package pubsub

import (
	"bytes"
	"context"

	"cloud.google.com/go/pubsub"
	"github.com/golang/protobuf/jsonpb"
	"go.chromium.org/luci/common/logging"
)

// ProcessPubsubMessage handles SourceRepoEvent pubsub message. If error is
// nil, the original message is acked and removed. Otherwise, the message will
// be available for consumptions again.
type ProcessPubsubMessage func(context.Context, *SourceRepoEvent) error

type pubsubReceiver interface {
	Receive(context.Context, func(context.Context, *pubsub.Message)) error
}

// NewClient initializes pubsub subscription.
func NewClient(ctx context.Context, gaeProject, subscriptionName string) (*pubsub.Subscription, error) {
	client, err := pubsub.NewClient(ctx, gaeProject)
	if err != nil {
		return nil, err
	}
	sub := client.Subscription(subscriptionName)
	return sub, nil
}

// Subscribe subscribes to pubsub and blocks until context is cancelled.
func Subscribe(ctx context.Context, sub pubsubReceiver,
	messageProcessor ProcessPubsubMessage) error {
	err := sub.Receive(ctx, func(ctx context.Context, m *pubsub.Message) {
		var event SourceRepoEvent
		err := jsonpb.Unmarshal(bytes.NewReader(m.Data), &event)
		if err != nil {
			logging.WithError(err).Errorf(
				ctx, "Error unmarshaling pubsub message")
			return
		}
		err = messageProcessor(ctx, &event)
		if err != nil {
			logging.WithError(err).Errorf(
				ctx, "Error processing pubsub message")
			return
		}
		m.Ack()
	})
	if err != nil {
		logging.Errorf(ctx, "Pubsub error: %s", err.Error())
	}
	return err
}
