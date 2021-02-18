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
	"fmt"
	"infra/chromeperf/pinpoint"
	"sync"

	"github.com/coreos/go-oidc/oidc"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
)

// pinpointClientFactory encapsulates the dialing and caching of a Pinpoint
// backend. Use newPinpointClientFactory to instantiate this type.
type pinpointClientFactory struct {
	mu             sync.Mutex
	Endpoint       string
	TCache         *tokenCache
	TLSCredentials credentials.TransportCredentials
	cachedClient   pinpoint.PinpointClient
	connection     *grpc.ClientConn
}

// Client returns a cached PinpointClient or newly dials a Pinpoint backend.
func (p *pinpointClientFactory) Client(ctx context.Context) (pinpoint.PinpointClient, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cachedClient != nil {
		return p.cachedClient, nil
	}

	opts := []grpc.DialOption{grpc.WithInsecure()}
	logSuffix := " (in insecure mode)"
	if p.TCache != nil || p.TLSCredentials != nil {
		logSuffix = ""
		v, o, err := newTokenVerifierAndConfig(ctx, p.TCache, interactiveFlowProvider{})
		if err != nil {
			return nil, errors.Annotate(err, "failed to get user token").Err()
		}

		jc, err := newJWTCredentials(p.TCache, v, o)
		if err != nil {
			return nil, errors.Annotate(err, "failed acquiring credentials").Err()
		}
		opts = []grpc.DialOption{
			grpc.WithTransportCredentials(p.TLSCredentials),
			grpc.WithPerRPCCredentials(jc),
		}
	}
	logging.Infof(ctx, "Connecting to %s%s", p.Endpoint, logSuffix)
	conn, err := grpc.Dial(p.Endpoint, opts...)
	if err != nil {
		return nil, err
	}

	p.connection = conn
	p.cachedClient = pinpoint.NewPinpointClient(conn)
	return p.cachedClient, nil
}

// Close should be used with a defer function which will acquire the
// appropriate locks when cleaning up.
func (p *pinpointClientFactory) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cachedClient = nil
	if p.connection != nil {
		return p.connection.Close()
	}
	return nil
}

// If tCache and tlsCreds are both nil, an "insecure" gRPC channel is created;
// this is useful for local testing.
func newPinpointClientFactory(endpoint string, tCache *tokenCache, tlsCreds credentials.TransportCredentials) *pinpointClientFactory {
	return &pinpointClientFactory{
		Endpoint:       endpoint,
		TCache:         tCache,
		TLSCredentials: tlsCreds,
	}
}

var authScopes = []string{oidc.ScopeOpenID, "email"}

const (
	providerURL          = "https://accounts.google.com"
	pinpointClientID     = "62121018386-kbva23qhdraiklj3ksdc8lams93q4gk6.apps.googleusercontent.com"
	pinpointClientSecret = "SUeCohQkddFi6gs737yE1k_y"
)

func newOAuth2Config(endpoint oauth2.Endpoint) *oauth2.Config {
	return &oauth2.Config{
		// We're hard-coding the ClientID for the Chromeperf Dashboard service here,
		// since we're the first one using OpenID Connect and we are looking only
		// for end-user authentication. This client ID only has very limited scopes
		// it requires, particularly just for getting the identity of a user.
		ClientID:     pinpointClientID,
		ClientSecret: pinpointClientSecret,
		Endpoint:     endpoint,
		Scopes:       authScopes,
		// This is a special URI which tells the provider that we're getting a token
		// out-of-band.
		RedirectURL: "urn:ietf:wg:oauth:2.0:oob",
	}
}

type tokenProvider interface {
	GetToken(context.Context, *oauth2.Config) (*oauth2.Token, error)
}

type interactiveFlowProvider struct {
}

func (interactiveFlowProvider) GetToken(ctx context.Context, c *oauth2.Config) (*oauth2.Token, error) {
	url := c.AuthCodeURL("", oauth2.AccessTypeOffline)
	fmt.Printf("Please visit the URL to obtain a code:\n\n%v\n\nThen enter the code provided: ", url)

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		return nil, errors.Annotate(err, "failed obtaining code from stdin").Err()
	}

	// Here we'll get the code and exchange it for an oauth2 token.
	t, err := c.Exchange(ctx, code)
	if err != nil {
		return nil, errors.Annotate(err, "failed exchanging code for an oauth2 token").Err()
	}
	return t, nil
}

// getUserToken retrieves and/or stores an OpenID Connect token id in JSON Web Token (JWT) form.
//
// This function will use cached tokens as much as possible, but if one's near
// expiration it will attempt to refresh that token from the providerUrl.
func newTokenVerifierAndConfig(
	ctx context.Context,
	tCache *tokenCache,
	tp tokenProvider,
) (*oidc.IDTokenVerifier, *oauth2.Config, error) {
	p, err := oidc.NewProvider(ctx, providerURL)
	if err != nil {
		return nil, nil, errors.Annotate(err, "failed to reach provider %q", providerURL).Err()
	}

	// From here we'll get an ID token using a combination of oidc and the standard oauth2 package.
	c := newOAuth2Config(p.Endpoint())
	v := p.Verifier(&oidc.Config{ClientID: pinpointClientID})

	// Check the cache for the token and re-use if possible. Continue on if the
	// token in the cache is no longer valid.
	if tCache != nil {
		_, _, err := tCache.GetVerifiedOrRefreshedToken(ctx, v, c)
		if err == nil {
			return v, c, nil
		}
	}

	t, err := tp.GetToken(ctx, c)
	if err != nil {
		return nil, nil, err
	}

	// Let's pull out the ID token from the response.
	rt, ok := t.Extra("id_token").(string)
	if !ok {
		return nil, nil, errors.Reason("failed to find an id_token in the response").Err()
	}

	_, err = v.Verify(ctx, rt)
	if err != nil {
		return nil, nil, errors.Annotate(err, "failed verifying retrieved id_token").Err()
	}

	// Before continuing, we should update the cache, and log errors.
	if tCache != nil {
		if _, _, err := tCache.UpdateToken(ctx, t, v); err != nil {
			logging.Errorf(ctx, "failed updating token cache: %q", err)
		}
	}
	return v, c, nil
}
