// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package pbutil contains methods for manipulating Weetbix protos.
package pbutil

import (
	"regexp"
	"regexp/syntax"
	"strings"

	"go.chromium.org/luci/common/errors"

	pb "infra/appengine/weetbix/proto/v1"
)

var (
	// Unspecified is the error to be used when something is unpeicified when it's
	// supposed to.
	Unspecified = errors.Reason("unspecified").Err()

	// DoesNotMatch is the error to be used when a string does not match a regex.
	DoesNotMatch = errors.Reason("does not match").Err()
)

// validateRegexp returns a non-nil error if re is an invalid regular
// expression.
func validateRegexp(re string) error {
	// Note: regexp.Compile uses syntax.Perl.
	if _, err := syntax.Parse(re, syntax.Perl); err != nil {
		return err
	}

	// Do not allow ^ and $ in the regexp, because we need to be able to prepend
	// a pattern to the user-supplied pattern.
	if strings.HasPrefix(re, "^") {
		return errors.Reason("must not start with ^; it is prepended automatically").Err()
	}
	if strings.HasSuffix(re, "$") {
		return errors.Reason("must not end with $; it is appended automatically").Err()
	}

	return nil
}

// ValidateWithRe validates a value matches the given re.
func ValidateWithRe(re *regexp.Regexp, value string) error {
	if value == "" {
		return Unspecified
	}
	if !re.MatchString(value) {
		return DoesNotMatch
	}
	return nil
}

// ValidateStringPair returns an error if p is invalid.
func ValidateStringPair(p *pb.StringPair) error {
	if err := ValidateWithRe(stringPairKeyRe, p.Key); err != nil {
		return errors.Annotate(err, "key").Err()
	}
	if len(p.Key) > maxStringPairKeyLength {
		return errors.Reason("key length must be less or equal to %d", maxStringPairKeyLength).Err()
	}
	if len(p.Value) > maxStringPairValueLength {
		return errors.Reason("value length must be less or equal to %d", maxStringPairValueLength).Err()
	}
	return nil
}

// ValidateVariant returns an error if vr is invalid.
func ValidateVariant(vr *pb.Variant) error {
	for k, v := range vr.GetDef() {
		p := pb.StringPair{Key: k, Value: v}
		if err := ValidateStringPair(&p); err != nil {
			return errors.Annotate(err, "%q:%q", k, v).Err()
		}
	}
	return nil
}

// ValidateVariantPredicate returns a non-nil error if p is determined to be
// invalid.
func ValidateVariantPredicate(p *pb.VariantPredicate) error {
	switch pr := p.Predicate.(type) {
	case *pb.VariantPredicate_Equals:
		return errors.Annotate(ValidateVariant(pr.Equals), "equals").Err()
	case *pb.VariantPredicate_Contains:
		return errors.Annotate(ValidateVariant(pr.Contains), "contains").Err()
	case nil:
		return Unspecified
	default:
		panic("impossible")
	}
}

// ValidateEnum returns a non-nil error if the value is not among valid values.
func ValidateEnum(value int32, validValues map[int32]string) error {
	if _, ok := validValues[value]; !ok {
		return errors.Reason("invalid value %d", value).Err()
	}
	return nil
}

// ValidateAnalyzedTestVariantStatus returns a non-nil error if s is invalid
// for a test variant.
func ValidateAnalyzedTestVariantStatus(s pb.AnalyzedTestVariantStatus) error {
	if err := ValidateEnum(int32(s), pb.AnalyzedTestVariantStatus_name); err != nil {
		return err
	}
	return nil
}

// ValidateAnalyzedTestVariantPredicate returns a non-nil error if p is
// determined to be invalid.
func ValidateAnalyzedTestVariantPredicate(p *pb.AnalyzedTestVariantPredicate) error {
	if err := validateRegexp(p.GetTestIdRegexp()); err != nil {
		return errors.Annotate(err, "test_id_regexp").Err()
	}

	if p.GetVariant() != nil {
		if err := ValidateVariantPredicate(p.GetVariant()); err != nil {
			return errors.Annotate(err, "variant").Err()
		}
	}

	if p.GetStatus() == pb.AnalyzedTestVariantStatus_STATUS_UNSPECIFIED {
		return nil
	}
	if err := ValidateAnalyzedTestVariantStatus(p.Status); err != nil {
		return errors.Annotate(err, "status").Err()
	}
	return nil
}
