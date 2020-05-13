// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package hart

import (
	"context"
	"fmt"
	"sync"

	"cloud.google.com/go/pubsub"
	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/common/logging"
	fleet "infra/libs/fleet/protos/go"

	"infra/libs/cros/lab_inventory/datastore"
)

var instance *Hart // Instance of HaRT
var once sync.Once

// Hart is a reference to PubSub connection
type Hart struct {
	client         *pubsub.Client
	topic          *pubsub.Topic
	ProjectID      string
	TopicID        string
	SubscriptionID string
}

// GetInstance returns instance of Hart.
//
// Can also set project, topic and subscription, when calling for the first
// time through dest
func GetInstance(ctx context.Context, proj, top, sub string) (*Hart, error) {
	var hart *Hart
	if instance == nil {
		client, err := pubsub.NewClient(ctx, proj)
		if err != nil {
			return nil, fmt.Errorf("pubsub.NewClient: %v", err)
		}
		topic := client.Topic(top)
		hart = &Hart{
			client:         client,
			topic:          topic,
			ProjectID:      proj,
			TopicID:        top,
			SubscriptionID: sub,
		}
	}
	once.Do(func() {
		instance = hart
		go hart.subscribeRoutine(ctx)
	})
	return instance, nil
}

// subscribeRoutine runs a routine that receives any AssetInfo sent by HaRT.
func (h *Hart) subscribeRoutine(ctx context.Context) {
	sub := h.client.Subscription(h.SubscriptionID)
	cctx, cancel := context.WithCancel(ctx)
	defer func() {
		// Restart the go routine if there is an unexpected crash
		cancel()
		if err := recover(); err != nil {
			logging.Errorf(ctx, " PubSub subscribe %s, restarting", err)
		}
		go h.subscribeRoutine(ctx)
	}()
	sub.Receive(cctx, func(ctx context.Context, m *pubsub.Message) {
		defer m.Ack()
		var response fleet.AssetInfoResponse
		perr := proto.Unmarshal(m.Data, &response)
		if perr == nil {
			if response.GetRequestStatus() == fleet.RequestStatus_SUCCESS {
				datastore.AddAssetInfo(ctx, response.GetAssets())
			}
		} else {
			logging.Warningf(ctx, "Unable to decode message %v", m.Attributes)
		}
	})
}

// publish a message to the topic in Hart, Blocks until ack.
func (h *Hart) publish(ctx context.Context, ids []string) (serverID string, err error) {
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
