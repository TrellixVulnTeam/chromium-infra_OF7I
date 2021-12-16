// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"fmt"
	"net/http"

	"go.chromium.org/luci/common/retry/transient"
	"go.chromium.org/luci/server/router"
)

// Sent by pubsub.
// This struct is just convenient for unwrapping the json message.
// See https://source.chromium.org/chromium/infra/infra/+/main:luci/appengine/components/components/pubsub.py;l=178;drc=78ce3aa55a2e5f77dc05517ef3ec377b3f36dc6e.
type pubsubMessage struct {
	Message struct {
		Data []byte
	}
	Attributes map[string]interface{}
}

func processErr(ctx *router.Context, err error) string {
	if transient.Tag.In(err) {
		// Transient errors are 500 so that PubSub retries them.
		ctx.Writer.WriteHeader(http.StatusInternalServerError)
		return "transient-failure"
	} else {
		// Permanent failures are 200s so that PubSub does not retry them.
		ctx.Writer.WriteHeader(http.StatusOK)
		return "permanent-failure"
	}
}

func buildID(hostname string, id int64) string {
	return fmt.Sprintf("%s/%v", hostname, id)
}
