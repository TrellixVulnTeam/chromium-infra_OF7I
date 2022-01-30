// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rpc

import (
	"context"

	"go.chromium.org/luci/grpc/appstatus"

	"google.golang.org/grpc/codes"

	pb "infra/appengine/weetbix/proto/v1"
)

// Rules implements pb.RulesServer.
type Rules struct {
}

// NewRules returns a new pb.RulesServer.
func NewRules() pb.RulesServer {
	return &pb.DecoratedRules{
		Prelude:  commonPrelude,
		Service:  &Rules{},
		Postlude: commonPostlude,
	}
}

// Retrieves a rule.
func (*Rules) Get(ctx context.Context, req *pb.GetRuleRequest) (*pb.Rule, error) {
	return nil, appstatus.Error(codes.Unimplemented, "not implemented")
}

// Lists rules.
func (*Rules) List(ctx context.Context, req *pb.ListRulesRequest) (*pb.ListRulesResponse, error) {
	return nil, appstatus.Error(codes.Unimplemented, "not implemented")
}

// Creates a new rule.
func (*Rules) Create(ctx context.Context, req *pb.CreateRuleRequest) (*pb.Rule, error) {
	return nil, appstatus.Error(codes.Unimplemented, "not implemented")
}

// Updates a rule.
func (*Rules) Update(ctx context.Context, req *pb.UpdateRuleRequest) (*pb.Rule, error) {
	return nil, appstatus.Error(codes.Unimplemented, "not implemented")
}
