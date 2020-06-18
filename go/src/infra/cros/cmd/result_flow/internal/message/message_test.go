// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package message_test

import (
	"context"
	"infra/cros/cmd/result_flow/internal/message"
	"sort"
	"strconv"
	"testing"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/pubsub/pstest"
	"google.golang.org/api/option"
	"google.golang.org/grpc"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform/result_flow"

	. "github.com/smartystreets/goconvey/convey"
)

func TestMessage(t *testing.T) {
	fakeConfig := &result_flow.PubSubConfig{
		Project:              "test-project",
		Topic:                "test-topic",
		Subscription:         "test-sub",
		MaxReceivingMessages: 50,
	}
	ctx := context.Background()
	// Start a fake server running locally.
	srv := pstest.NewServer()
	defer srv.Close()

	setupPubSubServer(ctx, fakeConfig, newConnection(srv.Addr))

	Convey("Given a list of build IDs, subscriber can parse them", t, func() {
		var want []int64 = []int64{8878576942164330944, 8878576942164330945}
		for _, bID := range want {
			if err := message.PublishBuildID(ctx, bID, fakeConfig, option.WithGRPCConn(newConnection(srv.Addr))); err != nil {
				panic(err)
			}
		}
		mClient, err := message.NewClient(ctx, fakeConfig, option.WithGRPCConn(newConnection(srv.Addr)))
		if err != nil {
			panic(err)
		}
		msgs, err := mClient.PullMessages(ctx)
		if err != nil {
			panic(err)
		}
		got := message.ToBuildIDs(ctx, msgs)
		So(len(want), ShouldEqual, len(got))
		sort.Slice(want, func(i, j int) bool { return want[i] < want[j] })
		sort.Slice(got, func(i, j int) bool { return got[i] < got[j] })
		for i := 0; i < len(got); i++ {
			So(want[i], ShouldEqual, got[i])
		}

	})
}

func stringToInt64(s string) int64 {
	i, _ := strconv.ParseInt(s, 10, 64)
	return i
}

// Create a new connection to the server without using TLS.
func newConnection(addr string) *grpc.ClientConn {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	return conn
}

// Setup the fake Topic and Subscription in the local Pub/Sub server.
func setupPubSubServer(ctx context.Context, conf *result_flow.PubSubConfig, conn *grpc.ClientConn) {
	client, err := pubsub.NewClient(ctx, conf.Project, option.WithGRPCConn(conn))
	if err != nil {
		panic(err)
	}
	defer client.Close()
	topic, err := client.CreateTopic(ctx, conf.Topic)
	if err != nil {
		panic(err)
	}
	if _, err := client.CreateSubscription(ctx, conf.Subscription, pubsub.SubscriptionConfig{Topic: topic}); err != nil {
		panic(err)
	}
}
