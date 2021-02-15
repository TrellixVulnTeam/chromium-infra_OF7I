// Copyright 2020 The Chromium Authors.
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

	"github.com/coreos/go-oidc/oidc"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"golang.org/x/oauth2"
)

type jwtCredentials struct {
	tCache       *tokenCache
	verifier     *oidc.IDTokenVerifier
	oauth2Config *oauth2.Config
}

func (t *jwtCredentials) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	tok, _, err := t.tCache.GetVerifiedOrRefreshedToken(ctx, t.verifier, t.oauth2Config)
	if err != nil {
		logging.Get(ctx).Errorf("failed to get JWT credentials: %q", err)
		return nil, errors.Annotate(err, "retrieving id token from cache failed").Err()
	}
	return map[string]string{"Authorization": "Bearer " + string(tok)}, nil
}

func (*jwtCredentials) RequireTransportSecurity() bool { return true }

func newJWTCredentials(tc *tokenCache, v *oidc.IDTokenVerifier, o *oauth2.Config) (*jwtCredentials, error) {
	if tc == nil || v == nil || o == nil {
		return nil, errors.Reason("all args must not be nil").Err()
	}
	return &jwtCredentials{tCache: tc, verifier: v, oauth2Config: o}, nil
}
