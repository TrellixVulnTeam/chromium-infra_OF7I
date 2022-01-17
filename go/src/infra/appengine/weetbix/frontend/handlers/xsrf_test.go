// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/auth/authtest"
	"go.chromium.org/luci/server/auth/xsrf"
	"go.chromium.org/luci/server/secrets"
	"go.chromium.org/luci/server/secrets/testsecrets"
)

func TestXSRF(t *testing.T) {
	Convey("With Router", t, func() {
		ctx := context.Background()
		// For user identification and XSRF Tokens.
		ctx = authtest.MockAuthConfig(ctx)
		ctx = auth.WithState(ctx, &authtest.FakeState{
			Identity: "user:someone@example.com",
		})
		ctx = secrets.Use(ctx, &testsecrets.Store{})

		router := routerForTesting(ctx)

		Convey("Get", func() {
			get := func() *http.Response {
				request, err := http.NewRequest("GET", "/api/xsrfToken", nil)
				So(err, ShouldBeNil)

				response := httptest.NewRecorder()
				router.ServeHTTP(response, request)
				return response.Result()
			}

			response := get()
			So(response.StatusCode, ShouldEqual, 200)

			b, err := io.ReadAll(response.Body)
			So(err, ShouldBeNil)

			var responseBody struct{ Token string }
			So(json.Unmarshal(b, &responseBody), ShouldBeNil)

			// Returned XSRF token should be valid.
			So(xsrf.Check(ctx, responseBody.Token), ShouldBeNil)
		})
	})
}
