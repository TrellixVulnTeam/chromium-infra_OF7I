// Copyright 2021 The Chromium Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cli

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/oauth2"
)

type mockTokenProvider struct {
	token *oauth2.Token
}

func (m *mockTokenProvider) GetToken(ctx context.Context, c *oauth2.Config) (*oauth2.Token, error) {
	return m.token, nil
}

type mockFailingTokenProvider struct {
}

func (mockFailingTokenProvider) GetToken(ctx context.Context, c *oauth2.Config) (*oauth2.Token, error) {
	return nil, fmt.Errorf("mock always fails")
}

// The JWT here is a randomly created from https://jwt.io/ as:
//
// {
//   "alg": "HS256",
//   "typ": "JWT"
// }
// {
//   "iss": "https://accounts.google.com",
//   "sub": "1234567890",
//   "name": "John Doe",
//   "iat": 1613127450,
//   "exp": 1613227450,
//   "aud": "62121018386-kbva23qhdraiklj3ksdc8lams93q4gk6.apps.googleusercontent.com"
// }
//
// This is known to be invalid, because we are using the wrong
// algorithm and at some point in the future, will be expired.
const invalidIDToken = `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJodHRwczovL2FjY291bnRzLmdvb2dsZS5jb20iLCJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNjEzMTI3NDUwLCJleHAiOjE2MTMyMjc0NTAsImF1ZCI6IjYyMTIxMDE4Mzg2LWtidmEyM3FoZHJhaWtsajNrc2RjOGxhbXM5M3E0Z2s2LmFwcHMuZ29vZ2xldXNlcmNvbnRlbnQuY29tIn0.S94K88ue4HKSPRl_BdRbh76-0IWQEytbppmjtCjN9tA`

func TestTokenVerifierAndConfigFactory_EmptyCache(t *testing.T) {
	Convey("Given an empty token cache and create a verifier and config", t, func() {
		ctx := context.Background()
		td, err := ioutil.TempDir(os.TempDir(), "pinpoint-test-*")
		So(err, ShouldBeNil)
		defer os.RemoveAll(td)
		os.Setenv("PINPOINT_CACHE_DIR", td)
		defer os.Unsetenv("PINPOINT_CACHE_DIR")
		tc, err := newTokenCache(ctx, td)
		So(err, ShouldBeNil)
		So(tc, ShouldNotBeNil)
		Convey("When we get an valid OAuth token", func() {
			tp := &mockTokenProvider{token: func() *oauth2.Token {
				t := &oauth2.Token{
					AccessToken:  "",
					TokenType:    "",
					RefreshToken: "",
					Expiry:       time.Time{},
				}
				return t.WithExtra(map[string]interface{}{"id_token": invalidIDToken})
			}()}
			v, c, err := newTokenVerifierAndConfig(ctx, tc, tp)
			Convey("Then we get no verifier", func() {
				So(v, ShouldBeNil)
			})
			Convey("Nor an OAuth2 Config", func() {
				So(c, ShouldBeNil)
			})
			Convey("And an error", func() {
				So(err, ShouldNotBeNil)
			})
		})
	})
}

func TestTokenVerifierAndConfigFactory_NilTokenCache(t *testing.T) {
	t.Parallel()
	Convey("Given no token cache and create a verifier and config", t, func() {
		Convey("When we obtain an invalid token from the provider", func() {
			ctx := context.Background()
			tp := &mockTokenProvider{token: func() *oauth2.Token {
				t := &oauth2.Token{
					AccessToken:  "",
					TokenType:    "",
					RefreshToken: "",
					Expiry:       time.Time{},
				}
				return t.WithExtra(map[string]interface{}{"id_token": invalidIDToken})
			}()}
			v, c, err := newTokenVerifierAndConfig(ctx, nil, tp)
			Convey("Then we get no verifier", func() {
				So(v, ShouldBeNil)
			})
			Convey("Nor an OAuth2 Config", func() {
				So(c, ShouldBeNil)
			})
			Convey("But an error", func() {
				So(err, ShouldNotBeNil)
			})
		})
		Convey("When we fail to get a valid OAuth token", func() {
			ctx := context.Background()
			tp := mockFailingTokenProvider{}
			v, c, err := newTokenVerifierAndConfig(ctx, nil, tp)
			Convey("Then we get no verifier", func() {
				So(v, ShouldBeNil)
			})
			Convey("Nor an OAuth2 Config", func() {
				So(c, ShouldBeNil)
			})
			Convey("But get an error", func() {
				So(err, ShouldNotBeNil)
			})
		})
	})
}
