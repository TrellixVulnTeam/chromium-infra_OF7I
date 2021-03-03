// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package message_test

import (
	"context"
	"testing"

	"infra/cros/cmd/result_flow/internal/message"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/pubsub/pstest"
	"google.golang.org/api/option"
	"google.golang.org/grpc"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform/result_flow"

	. "github.com/smartystreets/goconvey/convey"
)

type received struct {
	buildID                 int64
	parentUID               string
	shouldPollForCompletion bool
}

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

	mClient, err := message.NewClient(ctx, fakeConfig, 2, option.WithGRPCConn(newConnection(srv.Addr)))
	if err != nil {
		panic(err)
	}

	cases := []struct {
		description string
		in          map[string]string
		expected    *received
	}{
		{
			"CTP message with Build ID only",
			map[string]string{
				message.BuildIDKey: "8878576942164330944",
			},
			&received{
				buildID: 8878576942164330944,
			},
		},
		{
			"CTP message sent after the execution step",
			map[string]string{
				message.BuildIDKey:                 "8878576942164330944",
				message.ShouldPollForCompletionKey: "True",
			},
			&received{
				buildID:                 8878576942164330944,
				shouldPollForCompletion: true,
			},
		},
		{
			"Test runner message with Build ID and Parent UID",
			map[string]string{
				message.BuildIDKey:   "8878576942164330945",
				message.ParentUIDKey: "TestPlanRun/foo/fake-test",
			},
			&received{
				buildID:   8878576942164330945,
				parentUID: "TestPlanRun/foo/fake-test",
			},
		},
		{
			"Test runner message sent after the execution step",
			map[string]string{
				message.BuildIDKey:                 "8878576942164330945",
				message.ParentUIDKey:               "TestPlanRun/foo/fake-test",
				message.ShouldPollForCompletionKey: "True",
			},
			&received{
				buildID:                 8878576942164330945,
				parentUID:               "TestPlanRun/foo/fake-test",
				shouldPollForCompletion: true,
			},
		},
		{
			"Message with explicit false ShouldPollForCompletion in its attribute",
			map[string]string{
				message.BuildIDKey:                 "8878576942164330945",
				message.ParentUIDKey:               "TestPlanRun/foo/fake-test",
				message.ShouldPollForCompletionKey: "False",
			},
			&received{
				buildID:                 8878576942164330945,
				parentUID:               "TestPlanRun/foo/fake-test",
				shouldPollForCompletion: false,
			},
		},
	}
	for _, c := range cases {
		Convey(c.description, t, func() {
			if err := message.PublishBuild(ctx, c.in, fakeConfig, option.WithGRPCConn(newConnection(srv.Addr))); err != nil {
				panic(err)
			}
			msgs, err := mClient.PullMessages(ctx)
			mClient.AckMessages(ctx, msgs)
			if err != nil {
				panic(err)
			}
			got := message.ExtractBuildIDMap(ctx, msgs)
			So(got, ShouldContainKey, c.expected.buildID)
			So(message.GetParentUID(got[c.expected.buildID]), ShouldEqual, c.expected.parentUID)
			So(message.ShouldPollForCompletion(got[c.expected.buildID]), ShouldEqual, c.expected.shouldPollForCompletion)
		})
	}
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
