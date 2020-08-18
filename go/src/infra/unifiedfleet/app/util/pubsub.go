package util

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
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
