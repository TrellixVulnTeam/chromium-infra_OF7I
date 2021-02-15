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
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/coreos/go-oidc/oidc"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"golang.org/x/oauth2"
)

type tokenBundle struct {
	AuthToken     oauth2.Token
	IDToken       string       `json:"id_token,omitempty"`
	ParsedIDToken oidc.IDToken `json:"id_token_parsed,omitempty"`
}

type tokenCache struct {
	// TODO(https://crbug.com/1175615): Add synchronization when we need it.
	cacheFile   string
	cachedToken tokenBundle
}

func writeToTempFile(fileName string, data []byte) (string, error) {
	f, err := ioutil.TempFile(filepath.Dir(fileName), "tmp-token-cache-*")
	if err != nil {
		return "", errors.Annotate(err, "failed creating temporary file").Err()
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return "", errors.Annotate(err, "failed writing data").Err()
	}
	if err := f.Close(); err != nil {
		return "", errors.Annotate(err, "failed closing file").Err()
	}
	return f.Name(), nil
}

func readOrCreate(filename string) ([]byte, error) {
	d, err := ioutil.ReadFile(filename)
	if err != nil {
		// Create the file with no contents.
		f, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0600)
		if err != nil {
			// This is potentially a race, or the file already exists. We should
			// fail with the error.
			return nil, err
		}
		f.Close()
		return d, nil
	}
	return d, nil
}

// NewTokenCache will check whether the directory exists.
func newTokenCache(ctx context.Context, dir string) (*tokenCache, error) {
	// Ensure that the directory for the fileName exists, otherwise attempt to
	// create it. For directories, allow read-write-execute for owners only.
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}

	// Now set up a lock file, just for safety.
	lf, err := os.OpenFile(
		filepath.Join(dir, ".cache-lock"),
		os.O_CREATE|os.O_EXCL|os.O_RDONLY,
		0600,
	)
	if err != nil {
		return nil, errors.Annotate(err, "failed creating the lock file").Err()
	}
	defer func() {
		// Attempt to delete the file and ignore the error.
		lf.Close()
		os.Remove(lf.Name())
	}()

	res := &tokenCache{
		cacheFile: filepath.Join(dir, "cached-token"),
	}
	contents, err := readOrCreate(res.cacheFile)
	if err != nil {
		return nil, err
	}
	if len(contents) > 0 {
		if err := json.Unmarshal(contents, &res.cachedToken); err != nil {
			logging.Get(ctx).Errorf("failed decoding cache; %s", err)
			// This is an invalid token, we should clear the cached token, but not fail.
			res.cachedToken = tokenBundle{}
		} else {
			if len(res.cachedToken.IDToken) == 0 {
				logging.Get(ctx).Errorf("failed decoding cache: missing id_token")
				// This is also an invalid token, we should clear the cached token but not fail.
				res.cachedToken = tokenBundle{}
			}
		}
	}
	return res, nil
}

type tokenVerifier interface {
	Verify(context.Context, string) (*oidc.IDToken, error)
}

type oauth2Config interface {
	TokenSource(context.Context, *oauth2.Token) oauth2.TokenSource
}

// GetToken returns the cached token iff it is still valid.
func (t *tokenCache) GetVerifiedOrRefreshedToken(
	ctx context.Context,
	v tokenVerifier,
	c oauth2Config) ([]byte, *oidc.IDToken, error) {
	cachedToken := &t.cachedToken
	cachedIDToken := []byte(t.cachedToken.IDToken)
	if cachedToken.ParsedIDToken.Expiry.Before(time.Now().Add(time.Minute * -5)) {
		// Attempt to refresh the token now.
		src := c.TokenSource(ctx, &cachedToken.AuthToken)
		newToken, err := src.Token()
		if err != nil {
			return nil, nil, errors.Annotate(err, "failed obtaining a token").Err()
		}
		if newToken.AccessToken != cachedToken.AuthToken.AccessToken {
			if _, ok := newToken.Extra("id_token").(string); !ok {
				return nil, nil, errors.Reason("no id_token found in refreshed access token").Err()
			}
			cit, pit, err := t.UpdateToken(ctx, newToken, v)
			if err != nil {
				return nil, nil, errors.Annotate(err, "failed updating token cache").Err()
			}
			cachedToken.AuthToken = *newToken
			cachedToken.IDToken = string(cit)
			cachedToken.ParsedIDToken = *pit
			cachedIDToken = cit
		}
	}

	return cachedIDToken, &cachedToken.ParsedIDToken, nil
}

func inplaceUpdateFile(filename string, data []byte) error {
	tmpFilename, err := writeToTempFile(filename, data)
	if err != nil {
		return err
	}
	return os.Rename(tmpFilename, filename)
}

// UpdateToken ensures that we're atomically updating the token cache file.
func (t *tokenCache) UpdateToken(ctx context.Context, token *oauth2.Token, v tokenVerifier) ([]byte, *oidc.IDToken, error) {
	// Check first whether the data is valid.
	if token == nil {
		return nil, nil, errors.Reason("token is nil").Err()
	}

	if v == nil {
		return nil, nil, errors.Reason("verifier is nil").Err()
	}

	// We need to have the token include an id_token for us.
	idToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, nil, errors.Reason("token does not include an id_token").Err()
	}

	// Parse the token we got.
	pit, err := v.Verify(ctx, idToken)
	if err != nil {
		return nil, nil, errors.Annotate(err, "token verification failed").Err()
	}

	cf := t.cacheFile

	// Encode the bundle as JSON.
	b := tokenBundle{
		AuthToken:     *token,
		IDToken:       idToken,
		ParsedIDToken: *pit,
	}
	jd, err := json.Marshal(&b)
	if err != nil {
		return nil, nil, errors.Annotate(err, "failed encoding as json").Err()
	}
	if err := inplaceUpdateFile(cf, jd); err != nil {
		return nil, nil, errors.Annotate(err, "failed updating token cache").Err()
	}

	t.cachedToken.AuthToken = *token
	t.cachedToken.IDToken = idToken
	t.cachedToken.ParsedIDToken = *pit
	return []byte(idToken), pit, nil
}
