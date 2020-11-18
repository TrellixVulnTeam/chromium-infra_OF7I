// Copyright 2020 The LUCI Authors.
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

package main

import (
	"context"
	"crypto/sha256"
	"encoding/asn1"
	"encoding/base64"
	"fmt"
	"math/big"
	"strings"

	cloudkms "cloud.google.com/go/kms/apiv1"
	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
)

// Signs a given input using CloudKMS with key stored at kePath.
func signAsymmetric(ctx context.Context, client *cloudkms.KeyManagementClient, keyPath string, input []byte) (string, error) {
	digest := sha256.New()
	if _, err := digest.Write(input); err != nil {
		return "", fmt.Errorf("failed to create digest of input: %v", err)
	}

	// Build the signing request.
	req := &kmspb.AsymmetricSignRequest{
		Name: keyPath,
		Digest: &kmspb.Digest{
			Digest: &kmspb.Digest_Sha256{
				Sha256: digest.Sum(nil),
			},
		},
	}

	resp, err := client.AsymmetricSign(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to sign digest: %v", err)
	}

	// At this time, this tool assumes that all signatures are on SHA-256
	// digests and all keys are EC_SIGN_P256_SHA256.

	// To keep it in JWT spec we need to update the signature.

	var parsedSig struct{ R, S *big.Int }
	_, err = asn1.Unmarshal(resp.Signature, &parsedSig)
	if err != nil {
		return "", fmt.Errorf("failed to parse ecdsa signature bytes: %+v", err)
	}

	rBytes := parsedSig.R.Bytes()
	rBytesPadded := make([]byte, 32)
	copy(rBytesPadded[32-len(rBytes):], rBytes)

	sBytes := parsedSig.S.Bytes()
	sBytesPadded := make([]byte, 32)
	copy(sBytesPadded[32-len(sBytes):], sBytes)

	resp.Signature = append(rBytesPadded, sBytesPadded...)

	return encodeSegment(resp.Signature), nil
}

// Encodes JWT specific base64url encoding with padding stripped
func encodeSegment(seg []byte) string {
	return strings.TrimRight(base64.URLEncoding.EncodeToString(seg), "=")
}
