// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	bbv1 "go.chromium.org/luci/common/api/buildbucket/buildbucket/v1"
	"go.chromium.org/luci/server/router"

	. "github.com/smartystreets/goconvey/convey"
)

func makeReq(build bbv1.LegacyApiCommonBuildMessage) io.ReadCloser {
	bmsg := struct {
		Build    bbv1.LegacyApiCommonBuildMessage `json:"build"`
		Hostname string                           `json:"hostname"`
	}{build, "hostname"}
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

func TestHandleBuild(t *testing.T) {
	t.Parallel()

	Convey(`Test BuildbucketPubSubHandler`, t, func() {
		c := context.Background()
		buildExp := bbv1.LegacyApiCommonBuildMessage{
			Project: "fake",
			Bucket:  "luci.fake.bucket",
			Id:      87654321,
			Status:  bbv1.StatusCompleted,
		}
		h := httptest.NewRecorder()
		r := &http.Request{Body: makeReq(buildExp)}
		err := BuildbucketPubSubHandler(&router.Context{
			Context: c,
			Writer:  h,
			Request: r,
		})
		So(err, ShouldBeNil)
	})
}
