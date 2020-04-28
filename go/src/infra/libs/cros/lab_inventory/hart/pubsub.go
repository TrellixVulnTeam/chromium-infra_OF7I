// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package hart

import (
	"context"
	"fmt"
	"sync"

	fleet "infra/libs/fleet/protos/go"

	"cloud.google.com/go/pubsub"
	"github.com/golang/protobuf/proto"
)

var instance *Hart // Instance of HaRT
var once sync.Once
var projectID string = "hardware-request-tracker"
var topicID string = "assetInfoRequest"
var subID string = "AssetInfo"

// Hart is a reference to PubSub connection
type Hart struct {
	client *pubsub.Client
	topic  *pubsub.Topic
}

// GetInstance returns instance of Hart.
func GetInstance(ctx context.Context) (*Hart, error) {
	var hart *Hart
	if instance == nil {
		client, err := pubsub.NewClient(ctx, projectID)
		if err != nil {
			return nil, fmt.Errorf("pubsub.NewClient: %v", err)
		}
		topic := client.Topic(topicID)
		hart = &Hart{
			client: client,
			topic:  topic,
		}
	}
	once.Do(func() {
		instance = hart
	})
	return instance, nil
}

// Publish a message to the topic in Hart, Blocks until ack.
func (h *Hart) publish(ctx context.Context, ids []string) (string, error) {
	msg := &fleet.AssetInfoRequest{
		AssetTags: ids,
	}
	data, err := proto.Marshal(msg)
	if err != nil {
		return "", err
	}
	result := h.topic.Publish(ctx, &pubsub.Message{
		Data: data,
	})
	//Blocking until the result is returned
	return result.Get(ctx)
}

// SyncAssetInfoFromHaRT publishes the request for the ids to be synced.
// Returns server id response and error.
func (h *Hart) SyncAssetInfoFromHaRT(ctx context.Context, ids []string) (string, error) {
	return h.publish(ctx, ids)
}
