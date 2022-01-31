// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rpc

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/grpc/appstatus"
	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/span"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"infra/appengine/weetbix/internal/bugs"
	"infra/appengine/weetbix/internal/clustering"
	"infra/appengine/weetbix/internal/clustering/rules"
	"infra/appengine/weetbix/internal/config"
	"infra/appengine/weetbix/internal/config/compiledcfg"
	configpb "infra/appengine/weetbix/internal/config/proto"
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

var (
	RuleNameRe    = regexp.MustCompile(`^projects/(` + config.ProjectRePattern + `)/rules/(` + rules.RuleIDRePattern + `)$`)
	ProjectNameRe = regexp.MustCompile(`^projects/(` + config.ProjectRePattern + `)`)
)

// Retrieves a rule.
func (*Rules) Get(ctx context.Context, req *pb.GetRuleRequest) (*pb.Rule, error) {
	project, ruleID, err := parseRuleName(req.Name)
	if err != nil {
		return nil, validationError(err)
	}

	cfg, err := readProjectConfig(ctx, project)
	if err != nil {
		return nil, err
	}

	r, err := rules.Read(span.Single(ctx), project, ruleID)
	if err != nil {
		if err == rules.NotExistsErr {
			return nil, appstatus.Error(codes.NotFound, "rule does not exist")
		}
		// This will result in an internal error being reported to the caller.
		return nil, errors.Annotate(err, "reading rule %s", ruleID).Err()
	}
	return createRulePB(r, cfg.Config), nil
}

// Lists rules.
func (*Rules) List(ctx context.Context, req *pb.ListRulesRequest) (*pb.ListRulesResponse, error) {
	project, err := parseProjectName(req.Parent)
	if err != nil {
		return nil, validationError(err)
	}

	cfg, err := readProjectConfig(ctx, project)
	if err != nil {
		return nil, err
	}

	// TODO: Update to read all rules (not just active), and implement pagination.
	rs, err := rules.ReadActive(span.Single(ctx), project)
	if err != nil {
		// GRPCifyAndLog will log this, and report an internal error.
		return nil, errors.Annotate(err, "reading rules").Err()
	}

	rpbs := make([]*pb.Rule, 0, len(rs))
	for _, r := range rs {
		rpbs = append(rpbs, createRulePB(r, cfg.Config))
	}
	response := &pb.ListRulesResponse{
		Rules: rpbs,
	}
	return response, nil
}

// Creates a new rule.
func (*Rules) Create(ctx context.Context, req *pb.CreateRuleRequest) (*pb.Rule, error) {
	project, err := parseProjectName(req.Parent)
	if err != nil {
		return nil, validationError(err)
	}

	cfg, err := readProjectConfig(ctx, project)
	if err != nil {
		return nil, err
	}

	ruleID, err := rules.GenerateID()
	if err != nil {
		return nil, errors.Annotate(err, "generating Rule ID").Err()
	}
	user := auth.CurrentUser(ctx).Email

	r := &rules.FailureAssociationRule{
		Project:        project,
		RuleID:         ruleID,
		RuleDefinition: req.Rule.GetRuleDefinition(),
		BugID: bugs.BugID{
			System: req.Rule.Bug.GetSystem(),
			ID:     req.Rule.Bug.GetId(),
		},
		IsActive: req.Rule.GetIsActive(),
		SourceCluster: clustering.ClusterID{
			Algorithm: req.Rule.SourceCluster.GetAlgorithm(),
			ID:        req.Rule.SourceCluster.GetId(),
		},
	}

	if err := validateBugAgainstConfig(cfg, r.BugID); err != nil {
		return nil, validationError(err)
	}

	commitTime, err := span.ReadWriteTransaction(ctx, func(ctx context.Context) error {
		// Verify the bug is not used by another rule.
		bugRule, err := rules.ReadByBug(ctx, r.BugID)
		if err != nil && err != rules.NotExistsErr {
			return err
		}
		if err != rules.NotExistsErr {
			// Note: this validation could disclose the existence of rules
			// in projects other than those the user may have access to.
			// This is unavoidable in the context of the bug uniqueness
			// constraint we current have, which is needed to avoid Weetbix
			// making conflicting updates to the same bug.
			return validationError(fmt.Errorf("bug already used by another failure association rule (%s/%s)", bugRule.Project, bugRule.RuleID))
		}

		err = rules.Create(ctx, r, user)
		if err != nil {
			return validationError(err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	r.CreationTime = commitTime.In(time.UTC)
	r.CreationUser = user
	r.LastUpdated = commitTime.In(time.UTC)
	r.LastUpdatedUser = user

	return createRulePB(r, cfg.Config), nil
}

// Updates a rule.
func (*Rules) Update(ctx context.Context, req *pb.UpdateRuleRequest) (*pb.Rule, error) {
	project, ruleID, err := parseRuleName(req.Rule.GetName())
	if err != nil {
		return nil, validationError(err)
	}

	cfg, err := readProjectConfig(ctx, project)
	if err != nil {
		return nil, err
	}

	user := auth.CurrentUser(ctx).Email

	var updatedRule *rules.FailureAssociationRule
	f := func(ctx context.Context) error {
		rule, err := rules.Read(ctx, project, ruleID)
		if err != nil {
			if err == rules.NotExistsErr {
				return appstatus.Error(codes.NotFound, "rule does not exist")
			}
			// This will result in an internal error being reported to the
			// caller.
			return errors.Annotate(err, "read rule").Err()
		}
		if req.Etag != "" && ruleETag(rule) != req.Etag {
			// Attach a codes.Aborted appstatus to a vanilla error to avoid
			// ReadWriteTransaction interpreting this case for a scenario
			// in which it should retry the transaction.
			err := errors.New("etag mismatch")
			return appstatus.Attach(err, status.New(codes.Aborted, "the rule was modified since it was last read; the update was not applied."))
		}
		for _, path := range req.UpdateMask.Paths {
			// Only limited fields may be modified by the client.
			switch path {
			case "rule_definition":
				rule.RuleDefinition = req.Rule.RuleDefinition
			case "bug":
				bugID := bugs.BugID{
					System: req.Rule.Bug.GetSystem(),
					ID:     req.Rule.Bug.GetId(),
				}
				if err := validateBugAgainstConfig(cfg, bugID); err != nil {
					return validationError(err)
				}

				// Verify the bug is not used by another rule.
				bugRule, err := rules.ReadByBug(ctx, bugID)
				if err != nil && err != rules.NotExistsErr {
					// This will result in an internal error being reported
					// to the caller.
					return err
				}
				bugValid := (err == rules.NotExistsErr) || (bugRule.Project == project && bugRule.RuleID == ruleID)
				if !bugValid {
					// Note: this validation could disclose the existence of rules
					// in projects other than those the user may have access to.
					// This is unavoidable in the context of the bug uniqueness
					// constraint we current have, which is needed to avoid Weetbix
					// making conflicting updates to the same bug.
					return validationError(fmt.Errorf("bug already used by another failure association rule (%s/%s)", bugRule.Project, bugRule.RuleID))
				}

				rule.BugID = bugID
			case "is_active":
				rule.IsActive = req.Rule.IsActive
			default:
				return validationError(fmt.Errorf("unsupported field mask: %s", path))
			}
		}

		if err := rules.Update(ctx, rule, user); err != nil {
			return validationError(err)
		}
		updatedRule = rule
		return nil
	}
	commitTime, err := span.ReadWriteTransaction(ctx, f)
	if err != nil {
		return nil, err
	}
	updatedRule.LastUpdated = commitTime.In(time.UTC)
	updatedRule.LastUpdatedUser = user

	return createRulePB(updatedRule, cfg.Config), nil
}

// parseRuleName parses a rule resource name into its constituent ID parts.
func parseRuleName(name string) (project string, ruleID string, err error) {
	match := RuleNameRe.FindStringSubmatch(name)
	if match == nil {
		return "", "", errors.New("invalid rule name, expected format: projects/{project}/rules/{rule_id}")
	}
	return match[1], match[2], nil
}

// parseProjectName parses a project resource name into a project ID.
func parseProjectName(name string) (project string, err error) {
	match := ProjectNameRe.FindStringSubmatch(name)
	if match == nil {
		return "", errors.New("invalid project name, expected format: projects/{project}")
	}
	return match[1], nil
}

func createRuleName(project string, ruleID string) string {
	return fmt.Sprintf("projects/%s/rules/%s", project, ruleID)
}

func createRulePB(r *rules.FailureAssociationRule, cfg *configpb.ProjectConfig) *pb.Rule {
	return &pb.Rule{
		Name:           createRuleName(r.Project, r.RuleID),
		Project:        r.Project,
		RuleId:         r.RuleID,
		RuleDefinition: r.RuleDefinition,
		Bug:            createAssociatedBugPB(r.BugID, cfg),
		IsActive:       r.IsActive,
		SourceCluster: &pb.ClusterId{
			Algorithm: r.SourceCluster.Algorithm,
			Id:        r.SourceCluster.ID,
		},
		CreateTime:     timestamppb.New(r.CreationTime),
		CreateUser:     r.CreationUser,
		LastUpdateTime: timestamppb.New(r.LastUpdated),
		LastUpdateUser: r.LastUpdatedUser,
		Etag:           ruleETag(r),
	}
}

func ruleETag(rule *rules.FailureAssociationRule) string {
	return fmt.Sprintf(`W/"%s"`, rule.LastUpdated.UTC().Format(time.RFC3339Nano))
}

func createAssociatedBugPB(b bugs.BugID, cfg *configpb.ProjectConfig) *pb.AssociatedBug {
	// Fallback bug name and URL.
	linkText := fmt.Sprintf("%s/%s", b.System, b.ID)
	url := ""

	switch b.System {
	case bugs.MonorailSystem:
		project, id, err := b.MonorailProjectAndID()
		if err != nil {
			// Fallback.
			break
		}
		if project == cfg.Monorail.Project {
			if cfg.Monorail.DisplayPrefix != "" {
				linkText = fmt.Sprintf("%s/%s", cfg.Monorail.DisplayPrefix, id)
			} else {
				linkText = id
			}
		}
		if cfg.Monorail.MonorailHostname != "" {
			url = fmt.Sprintf("https://%s/p/%s/issues/detail?id=%s", cfg.Monorail.MonorailHostname, project, id)
		}
	default:
		// Fallback.
	}
	return &pb.AssociatedBug{
		System:   b.System,
		Id:       b.ID,
		LinkText: linkText,
		Url:      url,
	}
}

// readProjectConfig reads project config. This is intended for use in
// top-level RPC handlers. The caller should directly return an eny errors
// returned as the error of the RPC, the returned errors have been
// properly annotated with an appstatus.
func readProjectConfig(ctx context.Context, project string) (*compiledcfg.ProjectConfig, error) {
	cfg, err := compiledcfg.Project(ctx, project, time.Time{})
	if err != nil {
		if err == compiledcfg.NotExistsErr {
			return nil, validationError(errors.New("project does not exist in Weetbix"))
		}
		// GRPCifyAndLog will log this, and report an internal error to the caller.
		return nil, errors.Annotate(err, "obtain project config").Err()
	}
	return cfg, nil
}

// validateBugAgainstConfig validates the specified bug is consistent with
// the project configuration.
func validateBugAgainstConfig(cfg *compiledcfg.ProjectConfig, bug bugs.BugID) error {
	switch bug.System {
	case bugs.MonorailSystem:
		project, _, err := bug.MonorailProjectAndID()
		if err != nil {
			return err
		}
		if project != cfg.Config.Monorail.Project {
			return fmt.Errorf("bug not in expected monorail project (%s)", cfg.Config.Monorail.Project)
		}
	default:
		return fmt.Errorf("unsupported bug system: %s", bug.System)
	}
	return nil
}
