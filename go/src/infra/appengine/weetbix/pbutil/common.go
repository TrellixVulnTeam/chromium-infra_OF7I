// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package pbutil contains methods for manipulating Weetbix protos.
package pbutil

import (
	"fmt"
	"regexp"
	"sort"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"go.chromium.org/luci/common/errors"

	pb "infra/appengine/weetbix/proto/v1"
)

const maxStringPairKeyLength = 64
const maxStringPairValueLength = 256
const stringPairKeyPattern = `[a-z][a-z0-9_]*(/[a-z][a-z0-9_]*)*`

var stringPairKeyRe = regexp.MustCompile(fmt.Sprintf(`^%s$`, stringPairKeyPattern))
var stringPairRe = regexp.MustCompile(fmt.Sprintf("^(%s):(.*)$", stringPairKeyPattern))

// MustTimestampProto converts a time.Time to a *timestamppb.Timestamp and panics
// on failure.
func MustTimestampProto(t time.Time) *timestamppb.Timestamp {
	ts := timestamppb.New(t)
	if err := ts.CheckValid(); err != nil {
		panic(err)
	}
	return ts
}

// AsTime converts a *timestamppb.Timestamp to a time.Time.
func AsTime(ts *timestamppb.Timestamp) (time.Time, error) {
	if ts == nil {
		return time.Time{}, errors.Reason("unspecified").Err()
	}
	if err := ts.CheckValid(); err != nil {
		return time.Time{}, err
	}
	return ts.AsTime(), nil
}

func doesNotMatch(r *regexp.Regexp) error {
	return errors.Reason("does not match %s", r).Err()
}

// StringPair creates a pb.StringPair with the given strings as key/value field values.
func StringPair(k, v string) *pb.StringPair {
	return &pb.StringPair{Key: k, Value: v}
}

// StringPairs creates a slice of pb.StringPair from a list of strings alternating key/value.
//
// Panics if an odd number of tokens is passed.
func StringPairs(pairs ...string) []*pb.StringPair {
	if len(pairs)%2 != 0 {
		panic(fmt.Sprintf("odd number of tokens in %q", pairs))
	}

	strpairs := make([]*pb.StringPair, len(pairs)/2)
	for i := range strpairs {
		strpairs[i] = StringPair(pairs[2*i], pairs[2*i+1])
	}
	return strpairs
}

// StringPairFromString creates a pb.StringPair from the given key:val string.
func StringPairFromString(s string) (*pb.StringPair, error) {
	m := stringPairRe.FindStringSubmatch(s)
	if m == nil {
		return nil, doesNotMatch(stringPairRe)
	}
	return StringPair(m[1], m[3]), nil
}

// StringPairToString converts a StringPair to a key:val string.
func StringPairToString(pair *pb.StringPair) string {
	return fmt.Sprintf("%s:%s", pair.Key, pair.Value)
}

// StringPairsToStrings converts pairs to a slice of "{key}:{value}" strings
// in the same order.
func StringPairsToStrings(pairs ...*pb.StringPair) []string {
	ret := make([]string, len(pairs))
	for i, p := range pairs {
		ret[i] = StringPairToString(p)
	}
	return ret
}

// Variant creates a pb.Variant from a list of strings alternating
// key/value. Does not validate pairs.
// See also VariantFromStrings.
//
// Panics if an odd number of tokens is passed.
func Variant(pairs ...string) *pb.Variant {
	if len(pairs)%2 != 0 {
		panic(fmt.Sprintf("odd number of tokens in %q", pairs))
	}

	vr := &pb.Variant{Def: make(map[string]string, len(pairs)/2)}
	for i := 0; i < len(pairs); i += 2 {
		vr.Def[pairs[i]] = pairs[i+1]
	}
	return vr
}

// VariantFromStrings returns a Variant proto given the key:val string slice of its contents.
//
// If a key appears multiple times, the last pair wins.
func VariantFromStrings(pairs []string) (*pb.Variant, error) {
	if len(pairs) == 0 {
		return nil, nil
	}

	def := make(map[string]string, len(pairs))
	for _, p := range pairs {
		pair, err := StringPairFromString(p)
		if err != nil {
			return nil, errors.Annotate(err, "pair %q", p).Err()
		}
		def[pair.Key] = pair.Value
	}
	return &pb.Variant{Def: def}, nil
}

// SortedVariantKeys returns the keys in the variant as a sorted slice.
func SortedVariantKeys(vr *pb.Variant) []string {
	keys := make([]string, 0, len(vr.GetDef()))
	for k := range vr.GetDef() {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

var nonNilEmptyStringSlice = []string{}

// VariantToStrings returns a key:val string slice representation of the Variant.
// Never returns nil.
func VariantToStrings(vr *pb.Variant) []string {
	if len(vr.GetDef()) == 0 {
		return nonNilEmptyStringSlice
	}

	keys := SortedVariantKeys(vr)
	pairs := make([]string, len(keys))
	defMap := vr.GetDef()
	for i, k := range keys {
		pairs[i] = fmt.Sprintf("%s:%s", k, defMap[k])
	}
	return pairs
}

// VariantToStringPairs returns a slice of StringPair derived from *pb.Variant.
func VariantToStringPairs(vr *pb.Variant) []*pb.StringPair {
	defMap := vr.GetDef()
	if len(defMap) == 0 {
		return nil
	}

	keys := SortedVariantKeys(vr)
	sp := make([]*pb.StringPair, len(keys))
	for i, k := range keys {
		sp[i] = StringPair(k, defMap[k])
	}
	return sp
}
