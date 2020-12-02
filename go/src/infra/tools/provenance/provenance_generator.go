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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"strings"
	"time"

	cloudkms "cloud.google.com/go/kms/apiv1"
	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"

	provenancepb "infra/tools/provenance/proto"
)

// If unspecified, set expiry of token to 6 months.
const (
	DefaultExp int32 = 3600 * 24 * 30 * 6
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

// Prepares header segment of provenance meteadata.
func provenanceHeader(alg string, keyLocation string) (string, error) {
	header := &provenancepb.ProvenanceInfo_Header{
		Typ: "jwt",
		Alg: alg,
		Kid: keyLocation,
	}

	headerSegment, err := json.Marshal(header)

	if err != nil {
		return "", fmt.Errorf("failed to prepare header segment: %+v", err)
	}

	return encodeSegment(headerSegment), nil
}

// Prepares payload segment of provenance meteadata.
func provenacePayload(subjectHash string, topLevelSource *provenancepb.TopLevelSource, recipe string, exp int32) (string, error) {
	// int32 will not work after 03:14:07 UTC on 19 January 2038.
	epochNow := int32(time.Now().Unix())

	if exp == 0 {
		exp = epochNow + DefaultExp
	}

	buildEntryPoint := &provenancepb.BuildEntryPoint{
		Type:  "//bcid.corp.google.com/build_entry_point/luci/v1",
		Value: recipe,
	}

	builder := &provenancepb.ClaimPayload_Builder{
		Id: "//bcid.corp.google.com/builders/luci",
	}

	claim := &provenancepb.ClaimPayload{
		Builder:         builder,
		TopLevelSource:  topLevelSource,
		BuildEntryPoint: buildEntryPoint,
		SourceComplete:  false,
	}

	subject := &provenancepb.AttestedClaim_Subject{
		Sha256: subjectHash,
	}

	attestedClaim := &provenancepb.AttestedClaim{
		Type:         "//bcid.corp.google.com/attestations/core-provenance/v1",
		Subject:      subject,
		ClaimPayload: claim,
	}

	payload := &provenancepb.ProvenanceInfo_Payload{
		Aud:           "//binaryauthorization.googleapis.com/Attestation/v1",
		Iat:           epochNow,
		Exp:           exp,
		Nbf:           epochNow,
		AttestedClaim: attestedClaim,
	}

	payloadSegment, err := json.Marshal(payload)

	if err != nil {
		return "", fmt.Errorf("failed to prepare payload segment: %+v", err)
	}

	return encodeSegment(payloadSegment), nil
}

// Attestation struct for building/writing a single jwt.
type Attestation struct {
	jwt string
}

// Generates the final attestation and writes to a file.
func generateProvenance(ctx context.Context, client *cloudkms.KeyManagementClient, header string, body string, keyPath string, outfile string) (string, error) {
	signingInput := []byte(strings.Join([]string{header, body}, "."))
	provenanceSignature, err := signAsymmetric(ctx, client, keyPath, signingInput)
	if err != nil {
		return "", fmt.Errorf("failed to sign the provenance: %+v", err)
	}

	token := strings.Join([]string{header, body, provenanceSignature}, ".")
	rawAttestation := Attestation{token}

	provenance, err := json.Marshal(rawAttestation)
	if err != nil {
		return "", fmt.Errorf("failed to prepare provenance: %+v", err)
	}

	err = ioutil.WriteFile(outfile, provenance, 0744)
	if err != nil {
		return "", fmt.Errorf("failed to write provenance to file: %+v", err)
	}

	return token, nil
}
