// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"go.chromium.org/luci/common/errors"
)

var readCryptoHomeStatusInfoCases = []struct {
	testName       string
	rawOutput      string
	expectedCrypto *cryptoHomeStatus
	expectedErr    error
}{
	{
		"Input Contains TPM Info and TPM info complete",
		`
			{
				"installattrs": {
					"first_install": false,
					"initialized": false
				},
				"mounts": [  ],
				"tpm": {
					"being_owned": false,
					"can_connect": true,
					"can_decrypt": false,
					"can_encrypt": false,
					"can_load_srk": true,
					"can_load_srk_pubkey": true,
					"enabled": true,
					"has_context": true,
					"has_cryptohome_key": false,
					"has_key_handle": false,
					"last_error": 0,
					"owned": false,
					"srk_vulnerable_roca": false
				}
			}
		`,
		&cryptoHomeStatus{
			Tpm: map[string]interface{}{
				"being_owned":         interface{}(false),
				"can_connect":         interface{}(true),
				"can_decrypt":         interface{}(false),
				"can_encrypt":         interface{}(false),
				"can_load_srk":        interface{}(true),
				"can_load_srk_pubkey": interface{}(true),
				"enabled":             interface{}(true),
				"has_context":         interface{}(true),
				"has_cryptohome_key":  interface{}(false),
				"has_key_handle":      interface{}(false),
				"last_error":          interface{}(float64(0)),
				"owned":               interface{}(false),
				"srk_vulnerable_roca": interface{}(false),
			},
		},
		nil,
	},
	{
		"Input Contains TPM Info and TPM Info In Wrong Format",
		`
			{
				"tpm": {
					"being_owned": wrong format,
					"can_connect": true,
					"can_decrypt": false,
					"can_encrypt": false,
					"can_load_srk": true,
					"can_load_srk_pubkey": true,
					"enabled": true,
					"has_context": true,
					"has_cryptohome_key": false,
					"has_key_handle": false,
					"last_error": 0,
					"owned": false,
					"srk_vulnerable_roca": false
				}
			}
		`,
		nil,
		errors.Reason("read cryptohome status info: cannot read cryptohome status info").Err(),
	},
}

func TestReadCryptoHomeStatusInfo(t *testing.T) {
	ctx := context.Background()
	t.Parallel()
	for _, tt := range readCryptoHomeStatusInfoCases {
		tt := tt
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()
			actualCrypto, actualErr := ReadCryptoHomeStatusInfo(ctx, tt.rawOutput)
			if actualErr != nil && tt.expectedErr != nil {
				if !strings.Contains(actualErr.Error(), tt.expectedErr.Error()) {
					t.Errorf("Expected error %q, but got %q", tt.expectedErr, actualErr)
				}
			}
			if (actualErr == nil && tt.expectedErr != nil) || (actualErr != nil && tt.expectedErr == nil) {
				t.Errorf("Expected error %q, but got %q", tt.expectedErr, actualErr)
			}
			if !reflect.DeepEqual(actualCrypto, tt.expectedCrypto) {
				t.Errorf("Expected rw firmware version: %q, but got: %q", tt.expectedCrypto, actualCrypto)
			}
		})
	}
}
