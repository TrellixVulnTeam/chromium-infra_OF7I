// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package message

import (
	"context"
	"encoding/json"
	"fmt"
	"infra/libs/skylab/request"
	"strconv"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform/result_flow"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	pubsub "cloud.google.com/go/pubsub/apiv1"
	"google.golang.org/api/option"
	pubsubpb "google.golang.org/genproto/googleapis/pubsub/v1"
)

const (
	// BuildIDKey is the key to store Build ID in message attributes.
	BuildIDKey = "build_id"
	// ParentUIDKey is the key to store parent UID.
	ParentUIDKey = "parent_uid"
	// ShouldPollForCompletionKey is the key to store flag should_poll_for_completion.
	ShouldPollForCompletionKey = "should_poll_for_completion"
)

// PublishBuild publishes a Build to PubSub.
func PublishBuild(ctx context.Context, attr map[string]string, conf *result_flow.PubSubConfig, opts ...option.ClientOption) error {
	c, err := pubsub.NewPublisherClient(ctx, opts...)
	if err != nil {
		return fmt.Errorf("failed to create publisher client: %v", err)
	}
	defer c.Close()

	_, err = c.Publish(ctx, &pubsubpb.PublishRequest{
		Topic: fmt.Sprintf("projects/%s/topics/%s", conf.Project, conf.Topic),
		Messages: []*pubsubpb.PubsubMessage{
			{
				Attributes: attr,
			},
		},
	})
	return err
}

// Client defines an interface used to interact with pubsub
type Client interface {
	PullMessages(context.Context) ([]*pubsubpb.ReceivedMessage, error)
	AckMessages(context.Context, []*pubsubpb.ReceivedMessage) error
	Close() error
}

type messageClient struct {
	client       *pubsub.SubscriberClient
	subscription string
	maxMessages  int32
}

// NewClient creates a messageClient for PubSub subscriber.
func NewClient(c context.Context, conf *result_flow.PubSubConfig, batchSize int32, opts ...option.ClientOption) (Client, error) {
	client, err := pubsub.NewSubscriberClient(c, opts...)
	if err != nil {
		return nil, err
	}
	return &messageClient{
		client:       client,
		subscription: fmt.Sprintf("projects/%s/subscriptions/%s", conf.Project, conf.Subscription),
		maxMessages:  batchSize,
	}, nil
}

// PullMessages fetches messages from Pub/Sub.
func (m *messageClient) PullMessages(c context.Context) ([]*pubsubpb.ReceivedMessage, error) {
	req := pubsubpb.PullRequest{
		Subscription: m.subscription,
		MaxMessages:  m.maxMessages,
	}
	res, err := m.client.Pull(c, &req)
	if err != nil {
		return nil, errors.Annotate(err, "failed to pull messages").Err()
	}
	return res.ReceivedMessages, nil
}

// AckMessages acknowledges a list of messages.
func (m *messageClient) AckMessages(c context.Context, msgs []*pubsubpb.ReceivedMessage) error {
	if len(msgs) == 0 {
		return nil
	}
	AckIDs := make([]string, len(msgs))
	for i, msg := range msgs {
		AckIDs[i] = msg.AckId
	}
	return m.client.Acknowledge(c, &pubsubpb.AcknowledgeRequest{
		Subscription: m.subscription,
		AckIds:       AckIDs,
	})
}

// Close terminates the subscriber client.
func (m *messageClient) Close() error {
	return m.client.Close()
}

// ExtractBuildIDMap generates a map with key of Build ID and the value of parent UID.
func ExtractBuildIDMap(ctx context.Context, msgs []*pubsubpb.ReceivedMessage) map[int64]*pubsubpb.ReceivedMessage {
	m := make(map[int64]*pubsubpb.ReceivedMessage, len(msgs))
	for _, msg := range msgs {
		bID, err := extractBuildID(msg)
		if err != nil {
			logging.Errorf(ctx, "Failed to extract build ID, err: %v", err)
			continue
		}
		m[bID] = msg
	}
	return m
}

// ExtractParentUID extracts the parent request UID from the pubsub message.
func ExtractParentUID(msg *pubsubpb.ReceivedMessage) (string, error) {
	msgBody := struct {
		UserData string `json:"user_data"`
	}{}
	if err := json.Unmarshal(msg.GetMessage().GetData(), &msgBody); err != nil {
		return "", errors.Annotate(err, "could not parse pubsub message data").Err()
	}
	msgPayload := request.MessagePayload{}
	if err := json.Unmarshal([]byte(msgBody.UserData), &msgPayload); err != nil {
		return "", errors.Annotate(err, "could not extract Parent UID").Err()
	}
	return msgPayload.ParentRequestUID, nil
}

func extractBuildID(msg *pubsubpb.ReceivedMessage) (int64, error) {
	bID, err := strconv.ParseInt(msg.Message.Attributes[BuildIDKey], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("Failed to parse build ID from: %s", msg.Message.Attributes[BuildIDKey])
	}
	if bID == 0 {
		return 0, fmt.Errorf("Build ID can not be 0")
	}
	return bID, nil
}

// ShouldPollForCompletion returns true if the message contains the attribute
// "should_poll_for_completion" and its value is boolean true represented by
// a string.
func ShouldPollForCompletion(msg *pubsubpb.ReceivedMessage) bool {
	if v, ok := msg.Message.Attributes[ShouldPollForCompletionKey]; ok {
		b, err := strconv.ParseBool(v)
		return b && err == nil
	}
	return false
}
