// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Lex64 is a base64 encoding that preserves lexicographic ordering of bytestrings.
// It does this by using the alphanumeric characters, _, = and - as an alphabet. - is
// used as padding since it is the smallest character.
// The codec with padding preserves comparisons and has roundtrip equality.
// The codec without padding never reverses a comparison.
package lex64

import (
	"bytes"
	"encoding/base64"
	"io"
	"strings"
)

// The alphabet consists of the alphanumeric characters and = and _.
// The character '-' precedes all of these characters.
const alphabet = "0123456789=ABCDEFGHIJKLMNOPQRSTUVWXYZ_abcdefghijklmnopqrstuvwxyz"

// Lex64 uses '-' as padding. It appears before alphabet.
var lex64 = base64.NewEncoding(alphabet).WithPadding('-')

// Lex64NoPadding does not pad the output. Without padding, we get better space usage but worse algebraic properties.
// Without padding, no comparisons are reversed, but sort order is not preserved in general.
// Without padding, we don't have round-trip equivalence.
var lex64NoPadding = base64.NewEncoding(alphabet).WithPadding(base64.NoPadding)

// GetEncoding gets the encoding, parameterized by whether we want padding or not.
func getEncoding(padding bool) *base64.Encoding {
	if padding {
		return lex64
	}
	return lex64NoPadding
}

// Encode takes an array of bytes and converts it to a UTF-8 string encoded in lexicographic base64.
func Encode(src []byte, padding bool) (string, error) {
	buf := new(bytes.Buffer)
	encoder := base64.NewEncoder(getEncoding(padding), buf)
	_, err := encoder.Write(src)
	if err != nil {
		return "", err
	}
	if err := encoder.Close(); err != nil {
		return "", err
	}
	return string(buf.Bytes()), nil
}

// Decode takes a lexicographic base64 string and converts it to an array of bytes.
func Decode(encoded string, padding bool) ([]byte, error) {
	output := make([]byte, lex64.DecodedLen(len(encoded)))
	decoder := base64.NewDecoder(getEncoding(padding), strings.NewReader(encoded))
	n, err := decoder.Read(output)
	if err != nil && err != io.EOF {
		return nil, err
	}
	out := output[:n]
	return out, nil
}
