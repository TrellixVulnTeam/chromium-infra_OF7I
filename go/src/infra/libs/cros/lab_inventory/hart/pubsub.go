// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package hart

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"

	"cloud.google.com/go/pubsub"
	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/common/logging"
	fleet "infra/libs/fleet/protos/go"

	"infra/libs/cros/lab_inventory/datastore"
)

// PSRequest helps to unmarshall json data sent from pubsub
//
// The format of the data sent by PubSub is described in
// https://cloud.google.com/pubsub/docs/push?hl=en#receiving_push_messages
type PSRequest struct {
	Msg struct {
		Data      string `json:"data"`
		MessageID string `json:"messageId"`
	} `json:"message"`
	Sub string `json:"subscription"`
}

// PushHandler handles the pubsub push responses
//
// Decodes the response sent by PubSub and updates datastore. It doesn't
// return anything as required by https://cloud.google.com/pubsub/docs/push,
// this is because by default the return is 200 OK for http POST requests.
// It does not return any 4xx codes on error because it could lead to a loop
// where PubSub tries to push same message again which is rejected.
func PushHandler(ctx context.Context, r *http.Request) {
	// Decode request header
	var res PSRequest
	if err := json.NewDecoder(r.Body).Decode(&res); err != nil {
		logging.Errorf(ctx, "Unable to decode request JSON from pubsub %v", err)
		return
	}

	// Decode payload that contains the marshalled proto in base64
	data, err := base64.StdEncoding.DecodeString(res.Msg.Data)
	if err != nil {
		logging.Errorf(ctx, "Unable to decode payload data from pubsub %v", err)
		return
	}

	// Decode the proto contained in the payload
	var response fleet.AssetInfoResponse
	perr := proto.Unmarshal(data, &response)
	if perr == nil {
		if response.GetRequestStatus() == fleet.RequestStatus_OK {
			datastore.AddAssetInfo(ctx, response.GetAssets())
		}
		logging.Infof(ctx, "Status: %v", response.GetRequestStatus())
		missing := response.GetMissingAssetTags()
		logging.Infof(ctx, "Missing[%v]: %v", len(missing), missing)
		failed := response.GetFailedAssetTags()
		logging.Infof(ctx, "Failed[%v]: %v", len(failed), failed)
		logging.Infof(ctx, "Success reported for %v assets", len(response.GetAssets()))
	}
}

// publish a message to the topic in Hart, Blocks until ack.
func publish(ctx context.Context, topic *pubsub.Topic, ids []string) (serverID []string, err []error) {
	// Based on assets per second claim from HaRT
	// TODO(anushruth): Enforce 100 QPS
	batchSize := 100
	serverID = make([]string, (len(ids)/batchSize)+1)
	err = make([]error, (len(ids)/batchSize)+1)
	for i := 0; i < len(ids); i += batchSize {
		var msg *fleet.AssetInfoRequest
		if (i + batchSize) <= len(ids) {
			msg = &fleet.AssetInfoRequest{
				AssetTags: ids[i : i+batchSize],
			}
		} else {
			msg = &fleet.AssetInfoRequest{
				AssetTags: ids[i:],
			}
		}
		data, e := proto.Marshal(msg)
		if e != nil {
			serverID = append(serverID, "")
			err = append(err, e)
			continue
		}
		result := topic.Publish(ctx, &pubsub.Message{
			Data: data,
		})
		// Wait until the transaction is completed
		s, e := result.Get(ctx)
		serverID = append(serverID, s)
		err = append(err, e)
	}
	return
}

// SyncAssetInfoFromHaRT publishes the request for the ids to be synced.
func SyncAssetInfoFromHaRT(ctx context.Context, proj, topic string, ids []string) {
	client, err := pubsub.NewClient(ctx, proj)
	if err != nil {
		logging.Errorf(ctx, "pubsub.NewClient: %v", err)
		return
	}
	top := client.Topic(topic)
	servers, errs := publish(ctx, top, ids)
	for idx, e := range errs {
		if e != nil {
			logging.Errorf(ctx, "Error requesting asset info [%v]: %v", servers[idx], e)
		}
	}
}
