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
	"path/filepath"
	"testing"
	"time"

	"github.com/coreos/go-oidc/oidc"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/oauth2"
)

func TestTokenCacheCreation(t *testing.T) {
	t.Parallel()
	Convey("Creates a missing cache directory", t, func() {
		ctx := context.Background()
		td, err := ioutil.TempDir(os.TempDir(), "pinpoint-test-*")
		So(err, ShouldBeNil)
		defer os.RemoveAll(td)
		os.Setenv("PINPOINT_CACHE_DIR", td)
		defer os.Unsetenv("PINPOINT_CACHE_DIR")
		tc, err := newTokenCache(ctx, td)
		So(err, ShouldBeNil)
		So(tc, ShouldNotBeNil)
		So(tc.cacheFile, ShouldEqual, filepath.Join(td, "cached-token"))
	})
	Convey("Stale lock file fails token cache creation", t, func() {
		ctx := context.Background()
		td, err := ioutil.TempDir(os.TempDir(), "pinpoint-test-*")
		So(err, ShouldBeNil)
		defer os.RemoveAll(td)
		os.Setenv("PINPOINT_CACHE_DIR", td)
		defer os.Unsetenv("PINPOINT_CACHE_DIR")
		// Add a stale lock file in the directory.
		lfName := filepath.Join(td, ".cache-lock")
		lf, err := os.OpenFile(lfName, os.O_CREATE|os.O_EXCL|os.O_RDONLY, 0600)
		So(err, ShouldBeNil)
		defer lf.Close()
		defer os.Remove(lfName)
		tc, err := newTokenCache(ctx, td)
		So(err, ShouldNotBeNil)
		So(tc, ShouldBeNil)
	})
}

type mockVerifier struct {
	result *oidc.IDToken
	err    error
}

func (m *mockVerifier) Verify(context.Context, string) (*oidc.IDToken, error) {
	return m.result, m.err
}

func newMockVerifier(r *oidc.IDToken, e error) *mockVerifier {
	return &mockVerifier{result: r, err: e}
}

type mockAlwaysExpiringOAuth2Config struct{}

type mockTokenSource struct {
	expired bool
	t       *oauth2.Token
}

func (m *mockAlwaysExpiringOAuth2Config) TokenSource(ctx context.Context, t *oauth2.Token) oauth2.TokenSource {
	return &mockTokenSource{expired: true, t: t}
}

func (m *mockTokenSource) Token() (*oauth2.Token, error) {
	if m.expired {
		t := &oauth2.Token{
			AccessToken:  `eyJhY2Nlc3NfdG9rZW4iOiJzb21lLXRva2VuIiwiaWRfdG9rZW4iOiJzb21lLWlkLXRva2VuIn0=`,
			TokenType:    "Bearer",
			RefreshToken: `c29tZS1yZWZyZXNoLXRva2Vu`,
			Expiry:       time.Now(),
		}
		return t.WithExtra(map[string]interface{}{"id_token": "c29tZS10b2tlbg=="}), nil
	}
	return m.t, nil
}

type mockOAuth2ConfigTokenRandomizer struct{}

func (m *mockOAuth2ConfigTokenRandomizer) TokenSource(ctx context.Context, t *oauth2.Token) oauth2.TokenSource {
	return &mockTokenSource{expired: true, t: &oauth2.Token{}}
}

func TestTokenCacheFunctionality(t *testing.T) {
	// We need to set up an httpserver which will intercept the oidc provider requests, so that
	// we can respond appropriately.
	Convey("Given a valid token cache", t, func() {
		ctx := context.Background()
		td, err := ioutil.TempDir(os.TempDir(), "pinpoint-test-*")
		So(err, ShouldBeNil)
		defer os.RemoveAll(td)
		os.Setenv("PINPOINT_CACHE_DIR", td)
		defer os.Unsetenv("PINPOINT_CACHE_DIR")
		tc, err := newTokenCache(ctx, td)
		So(err, ShouldBeNil)
		So(tc, ShouldNotBeNil)
		Convey("When the cached token is expired", func() {
			tc.cachedToken.ParsedIDToken.Expiry.Add(-time.Hour)
			Convey("Then we attempt to get a new token", func() {
				v := newMockVerifier(&oidc.IDToken{
					Issuer:          "some-issuer",
					Audience:        []string{"some-audience"},
					Subject:         "some-user@example.com",
					Expiry:          time.Now().Add(time.Hour * 12),
					IssuedAt:        time.Now(),
					Nonce:           "",
					AccessTokenHash: "abcdef",
				}, nil)
				c := &mockAlwaysExpiringOAuth2Config{}
				_, t, err := tc.GetVerifiedOrRefreshedToken(ctx, v, c)
				So(err, ShouldBeNil)
				So(t, ShouldNotBeNil)
			})
		})
		Convey("When the cached token is invalid", func() {
			tc.cachedToken = tokenBundle{
				AuthToken: oauth2.Token{},
				IDToken:   "",
				ParsedIDToken: oidc.IDToken{
					Issuer:          "",
					Audience:        []string{},
					Subject:         "",
					Expiry:          time.Time{},
					IssuedAt:        time.Time{},
					Nonce:           "",
					AccessTokenHash: "",
				},
			}
			Convey("Then we are forced to get a new token", func() {
				v := newMockVerifier(nil, fmt.Errorf("mock: always invalid token"))
				c := &mockAlwaysExpiringOAuth2Config{}
				_, t, err := tc.GetVerifiedOrRefreshedToken(ctx, v, c)
				So(err, ShouldNotBeNil)
				So(t, ShouldBeNil)
			})
		})
	})
}
