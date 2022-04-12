// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package pubsub

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	bbv1 "go.chromium.org/luci/common/api/buildbucket/buildbucket/v1"
)

func TestBuildBucketPubsub(t *testing.T) {
	t.Parallel()

	Convey("Buildbucket Pubsub Handler", t, func() {
		c := context.Background()
		buildExp := bbv1.LegacyApiCommonBuildMessage{
			Project: "fake",
			Bucket:  "luci.fake.bucket",
			Id:      87654321,
			Status:  bbv1.StatusCompleted,
		}
		r := &http.Request{Body: makeBBReq(buildExp, "bb-hostname")}
		err := buildbucketPubSubHandlerImpl(c, r)
		So(err, ShouldBeNil)
	})
}

func makeBBReq(build bbv1.LegacyApiCommonBuildMessage, hostname string) io.ReadCloser {
	bmsg := struct {
		Build    bbv1.LegacyApiCommonBuildMessage `json:"build"`
		Hostname string                           `json:"hostname"`
	}{build, hostname}
	bm, _ := json.Marshal(bmsg)

	msg := struct {
		Message struct {
			Data []byte
		}
		Attributes map[string]interface{}
	}{struct{ Data []byte }{Data: bm}, nil}
	jmsg, _ := json.Marshal(msg)
	return ioutil.NopCloser(bytes.NewReader(jmsg))
}
