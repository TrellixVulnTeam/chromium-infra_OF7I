// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package filterexp

import (
	"reflect"

	"github.com/google/cel-go/cel"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"infra/cros/karte/internal/errors"
)

// Comparisons are the valid comparisons.
// TODO(gregorynisbet): Factor these into a symbols package or similar to isolate
//                      details about how CEL names infix operators.
var comparisons = map[string]bool{
	"_<_":  true,
	"_<=_": true,
	"_==_": true,
	"_>=_": true,
	"_>_":  true,
}

// IsComparison checks whether the function names a CEL comparison (e.g. _>_ and _==_).
func isComparison(name string) bool {
	return comparisons[name]
}

// An Expression is a constant (an int or a string)
// or an application consisting of head and a tail of expressions.
//
// Alternative #1: an expression is a constant.
// Alternative #2: an expression is a symbol.
// Alternative #3: an expression is a fixed function applied to
//                 a list of arguments.
type Expression interface {
	// IsExpression is a placeholder method for type safety.
	isExpression()
}

// A Constant is a Go value such as "foo" or 4 that represents the value of a CEL constant.
type Constant struct {
	Value interface{}
}

// IsExpression is a placeholder method for type safety.
func (c *Constant) isExpression() {}

// An identifier is a free variable in a filter expression.
type Identifier struct {
	Value string
}

// IsExpression is a placeholder method for type safety.
func (i *Identifier) isExpression() {}

// An application is a function application. This is the only kind of node that
// branches. Anything that takes arguments, whether a connective (&&), a predicate (<=), or
// a function, is treated as a function application.
type Application struct {
	Head string
	Tail []Expression
}

// IsExpression is a placeholder method for type safety.
func (a *Application) isExpression() {}

// Parse takes a program and produces a list of expressions.
// A program in this case is a query in a restricted subset of the CEL language.
// The only supported programs are comparisons (e.g. a == "foo") combined together using "&&".
// The expressions returned by parse should be interpreted as implicitly joined together
// with a variadic "and".
func Parse(program string) ([]Expression, error) {
	if program == "" {
		return nil, nil
	}
	env, err := cel.NewEnv()
	if err != nil {
		return nil, errors.Annotate(err, "parse program %q", program).Err()
	}
	ast, issues := env.Parse(program)
	if err := issues.Err(); err != nil {
		return nil, errors.Annotate(err, "parse program %q", program).Err()
	}

	var out []Expression

	hopper := []*exprpb.Expr{ast.Expr()}

	// As long as the hopper is non-empty, we grab the rightmost element of the hopper.
	// If it is an expression headed by "_&&_", we take its arguments and add them back to the hopper.
	// If the expression is headed by a comparison, we add that comparison to "out", the list of all
	// expressions. This kind of traversal results in the rightmost child of any AST node being considered
	// before the others temporally, so at the very end we will have to reverse the order of out.
	for len(hopper) != 0 {
		current := hopper[len(hopper)-1]
		hopper = hopper[0 : len(hopper)-1]
		v, ok := current.ExprKind.(*exprpb.Expr_CallExpr)
		if !ok {
			return nil, errors.Errorf("parse program: unexpected expression kind %q", reflect.TypeOf(current).Name())
		}
		c := v.CallExpr
		switch {
		case c.Function == "_&&_":
			var err error
			hopper, err = processConjunct(hopper, c)
			if err != nil {
				return nil, errors.Annotate(err, "parse program").Err()
			}
		case isComparison(c.Function):
			var err error
			item, err := processComparison(out, c)
			if err != nil {
				return nil, errors.Annotate(err, "parse program").Err()
			}
			out = append(out, item)
		default:
			return nil, errors.Errorf("parse program: unsupported top-level function %q", c.Function)
		}
	}

	// In order to preserve the linear order of the conjuncts, we must reverse the list.
	revOut := []Expression{}
	for i := len(out) - 1; i >= 0; i-- {
		revOut = append(revOut, out[i])
	}

	return revOut, nil
}

// NewIdentifier produces a new identifier expression.
func NewIdentifier(identifier string) Expression {
	return &Identifier{identifier}
}

// NewConstant produces a new constant expression.
func NewConstant(constant interface{}) Expression {
	return &Constant{constant}
}

// NewApplication produces a new application expression.
func NewApplication(head string, tail ...Expression) Expression {
	return &Application{
		Head: head,
		Tail: tail,
	}
}

// ProcessConjunct takes a list of expressions to analyze and a current expression that is headed by "_&&_".
// It then takes the arguments of this expression and adds them back to hopper.
//
// This function invalidates the argument hopper.
//
// It is intended to be called in the following way:
//
//   hopper, ... = processConjunct(hopper, ...)
//
func processConjunct(hopper []*exprpb.Expr, e *exprpb.Expr_Call) ([]*exprpb.Expr, error) {
	for _, item := range e.Args {
		if _, ok := item.ExprKind.(*exprpb.Expr_CallExpr); !ok {
			return nil, status.Errorf(codes.Internal, "process conjunct: not a call expression")
		}
		hopper = append(hopper, item)
	}
	return hopper, nil
}

// ProcessComparison takes a list of expressions to be emitted and a current expression that is headed by a
// a comparator. It then produces an expression and adds it to the list of expressions to be emitted.
//
// This function invalidates the argument conjuncts.
//
// It is intended to be called in the following way:
//
//   item, err := processComparison(out, ...)
//   ...
//   out := append(out, item)
//
func processComparison(comparisons []Expression, e *exprpb.Expr_Call) (Expression, error) {
	newExpr := Application{Head: e.Function}
	for _, item := range e.Args {
		switch v := item.ExprKind.(type) {
		case *exprpb.Expr_ConstExpr:
			val, err := extractConstantValue(v.ConstExpr)
			if err != nil {
				return nil, errors.Annotate(err, "process comparison").Err()
			}
			newExpr.Tail = append(newExpr.Tail, val)
		case *exprpb.Expr_IdentExpr:
			newExpr.Tail = append(newExpr.Tail, NewIdentifier(v.IdentExpr.Name))
		default:
			return nil, errors.Errorf("process comparison: unknown argument type %q", reflect.TypeOf(item).Name())
		}
	}
	return &newExpr, nil
}

// ExtractConstantValue extracts the golang value from a constant and ensure that it has
// a supported type.
func extractConstantValue(e *exprpb.Constant) (Expression, error) {
	switch v := e.ConstantKind.(type) {
	case *exprpb.Constant_StringValue:
		return NewConstant(v.StringValue), nil
	default:
		return nil, errors.Errorf("extract constant value: type %q not implemented", reflect.TypeOf(v).Name())
	}
}
