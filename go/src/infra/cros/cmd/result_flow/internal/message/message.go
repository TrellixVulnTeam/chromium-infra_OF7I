// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package message

import (
	"context"
	"fmt"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform/result_flow"

	pubsub "cloud.google.com/go/pubsub/apiv1"
	pubsubpb "google.golang.org/genproto/googleapis/pubsub/v1"
)

// BuildIDKeyName is the key name to store Build ID in message attributes.
const BuildIDKeyName = "build_id"

// PublishBuildID publishes a Build ID to PubSub.
func PublishBuildID(ctx context.Context, bID int64, conf *result_flow.PubSubConfig) error {
	c, err := pubsub.NewPublisherClient(ctx)
	defer c.Close()
	if err != nil {
		return fmt.Errorf("failed to create publisher client: %v", err)
	}

	_, err = c.Publish(ctx, &pubsubpb.PublishRequest{
		Topic: fmt.Sprintf("projects/%s/topics/%s", conf.Project, conf.Topic),
		Messages: []*pubsubpb.PubsubMessage{
			genPublishRequest(bID),
		},
	})
	return err
}

func genPublishRequest(bID int64) *pubsubpb.PubsubMessage {
	return &pubsubpb.PubsubMessage{
		Attributes: map[string]string{
			BuildIDKeyName: fmt.Sprintf("%d", bID),
		},
	}
}
