// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package filterexp

import (
	"go.chromium.org/luci/gae/service/datastore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"infra/cros/karte/internal/errors"
)

// ApplyConditions takes a datastore query and a list of conditions to impose
// and returns a modified query with those conditions imposed. A condition has
// the form "field COMPARATOR value" (e.g. `kind == "a"`).
//
// sample usage:
//
//   ApplyConditions(
//     q,
//     []Expresson{
//       NewApplication(
//        "_==_",
//        NewIdentifier("a"),
//        NewConstant("b"),
//     },
//   )
//
func ApplyConditions(q *datastore.Query, conditions []Expression) (*datastore.Query, error) {
	if conditions == nil {
		return q, nil
	}

	for _, expr := range conditions {
		r, err := validateComparison(expr)
		if err != nil {
			return nil, errors.Annotate(err, "apply conditions").Err()
		}

		switch r.comparator {
		case "_==_":
			q = q.Eq(r.field, r.value)
		default:
			return nil, errors.Errorf("apply conditions: comparator %q not yet implemented", r.comparator)
		}
	}

	return q, nil
}

// ComparisonParseResult is the result of parsing a single comparison. It has a comparator
// such as _==_ or _<_, a field, and a value, which is the value of a constant.
//
// TODO(gregorynisbet): Change ComparisonParseResult to use an ApplyToQuery method.
type comparisonParseResult struct {
	comparator string
	field      string
	// Supported types: string
	value interface{}
}

// ValidateComparison takes an expression that should be a comparison, confirms that it really is
// a comparison, and extracts information from the expression AST.
func validateComparison(e Expression) (*comparisonParseResult, error) {
	if e == nil {
		return nil, status.Errorf(codes.Internal, "validate comparison: expression is nil")
	}

	appl, ok := e.(*Application)
	if !ok {
		return nil, status.Errorf(codes.Internal, "validate comparison: expression is not application")
	}

	if len(appl.Tail) > 2 {
		return nil, status.Errorf(codes.Internal, "validate comparison: arity %d is too small", len(appl.Tail))
	}

	fieldExpr, ok := appl.Tail[0].(*Identifier)
	if !ok {
		return nil, status.Errorf(codes.Internal, "validate comparison: field is not identifier")
	}
	field := fieldExpr.Value

	valueExpr, ok := appl.Tail[1].(*Constant)
	if !ok {
		return nil, status.Errorf(codes.Internal, "validate comparison: value is not constant")
	}
	value := valueExpr.Value

	v, ok := value.(string)
	if !ok {
		return nil, status.Errorf(codes.Internal, "validate comparison: constant type not yet implemented")
	}

	return &comparisonParseResult{
		comparator: appl.Head,
		field:      field,
		value:      v,
	}, nil
}
