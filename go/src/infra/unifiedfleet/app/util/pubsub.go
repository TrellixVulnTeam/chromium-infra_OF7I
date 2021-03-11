package util

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"

	"cloud.google.com/go/pubsub"
	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/common/errors"

	ufspb "infra/unifiedfleet/api/v1/models"
	"infra/unifiedfleet/app/config"
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

// NewPSRequest returns a PSRequest object constructed from http request
func NewPSRequest(r *http.Request) (*PSRequest, error) {
	var res PSRequest
	if err := json.NewDecoder(r.Body).Decode(&res); err != nil {
		return nil, err
	}
	return &res, nil
}

// DecodeMessage decodes []byte pertaining to the request
func (p *PSRequest) DecodeMessage() ([]byte, error) {
	var err error
	if data, err := base64.StdEncoding.DecodeString(p.Msg.Data); err == nil {
		return data, err
	}
	return nil, err
}

// PublishHaRTAssetInfoRequest sends a request for asset info update for given assets
func PublishHaRTAssetInfoRequest(ctx context.Context, assets []string) (err error) {
	// Create a pubsub topic for publish to HaRT for update
	conf := config.Get(ctx).GetHart()
	batchSize := int(conf.GetBatchSize()) // Convert uint32 to int.
	if batchSize == 0 {
		return errors.Reason("PublishHaRTAssetInfoRequest - batch_size cannot be 0. Set correct configuration").Err()
	}
	proj := conf.GetProject()
	topic := conf.GetTopic()
	client, err := pubsub.NewClient(ctx, proj)
	var top *pubsub.Topic
	if err != nil {
		return errors.Annotate(err, "PublishHaRTAssetInfoRequest - Failed to publish asset info update request to %s", proj).Err()
	}
	// Return resources at the end of request
	defer client.Close()

	top = client.Topic(topic)
	// Topic's publish methods creates a few go routines. Call stop before returning.
	defer top.Stop()

	for i := 0; i < len(assets); i += batchSize {
		msg := &ufspb.AssetInfoRequest{}
		if (i + batchSize) <= len(assets) {
			msg.AssetTags = assets[i : i+batchSize]
		} else {
			msg.AssetTags = assets[i:]
		}
		data, e := proto.Marshal(msg)
		if e != nil {
			// Append error messages to a single error.
			err = errors.Annotate(err, "Failed to marshal message %v: %v", msg, e.Error()).Err()
			continue
		}
		result := top.Publish(ctx, &pubsub.Message{
			Data: data,
		})
		// Wait until the transaction is completed
		s, e := result.Get(ctx)
		if e != nil {
			// Append error messages to a single error.
			err = errors.Annotate(err, "PubSub req %v failed: %s", s, e.Error()).Err()
			continue
		}
	}
	return
}
