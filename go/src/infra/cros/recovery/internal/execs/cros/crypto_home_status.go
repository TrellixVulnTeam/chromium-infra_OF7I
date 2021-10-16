// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"encoding/json"
	"strings"

	"go.chromium.org/luci/common/errors"
)

// cryptoHomeStatus holds info about the tpm(Trusted Platform Module specification)
// portion of the "cryptohome --action=status"
type cryptoHomeStatus struct {
	Tpm map[string]interface{}
}

// ReadCryptoHomeStatusInfo read and parse TPM status information
// and return cryptoHomeStatus struct to access this information.
// The cryptohome command emits status information in JSON format. It looks something like this:
//  {
//     "installattrs": {
//        ...
//     },
//     "mounts": [ {
//        ...
//     } ],
//     "tpm": {
//        "being_owned": false,
//        "can_connect": true,
//        "can_decrypt": false,
//        "can_encrypt": false,
//        "can_load_srk": true,
//        "can_load_srk_pubkey": true,
//        "enabled": true,
//        "has_context": true,
//        "has_cryptohome_key": false,
//        "has_key_handle": false,
//        "last_error": 0,
//        "owned": true
//     }
//  }
func ReadCryptoHomeStatusInfo(ctx context.Context, rawOutput string) (*cryptoHomeStatus, error) {
	crypto := &cryptoHomeStatus{}
	output := strings.TrimSpace(rawOutput)
	if err := json.Unmarshal([]byte(output), crypto); err != nil {
		return nil, errors.Annotate(err, "read cryptohome status info: cannot read cryptohome status info").Err()
	}
	return crypto, nil
}

// ReadTPMBool read tpm field value and convert it into golang type bool value.
func (crypto *cryptoHomeStatus) ReadTPMBool(tpmField string) (bool, bool) {
	value, ok := crypto.Tpm[tpmField]
	if !ok {
		return false, false
	}
	boolValue, okToConvert := value.(bool)
	return boolValue, okToConvert
}

// ReadFloat64FromTPMField read tpm field value and convert it into golang type float64 value.
func (crypto *cryptoHomeStatus) ReadTPMFloat64(tpmField string) (float64, bool) {
	value, ok := crypto.Tpm[tpmField]
	if !ok {
		return -1, false
	}
	floatValue, okToConvert := value.(float64)
	return floatValue, okToConvert
}
