// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"google.golang.org/genproto/protobuf/field_mask"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/auth/xsrf"
	"go.chromium.org/luci/server/router"
	"go.chromium.org/luci/server/span"

	"infra/appengine/weetbix/internal/bugs"
	"infra/appengine/weetbix/internal/clustering/rules"
	configpb "infra/appengine/weetbix/internal/config/proto"
)

// ValidationError is the tag applied to validation errors.
var ValidationErrorTag = errors.BoolTag{Key: errors.NewTagKey("validation error")}

// concurrentModification is an error that is returned if a concurrent
// update is detected.
var ConcurrentModificationTag = errors.BoolTag{Key: errors.NewTagKey("concurrent modification")}

// ListRules serves a GET request for
// /api/projects/:project/rules.
func (h *Handlers) ListRules(ctx *router.Context) {
	transctx, cancel := span.ReadOnlyTransaction(ctx.Context)
	defer cancel()

	projectID, ok := obtainProjectOrError(ctx)
	if !ok {
		return
	}
	rs, err := rules.ReadActive(transctx, projectID)
	if err != nil {
		logging.Errorf(ctx.Context, "Reading rules: %s", err)
		http.Error(ctx.Writer, "Internal server error.", http.StatusInternalServerError)
		return
	}

	respondWithJSON(ctx, rs)
}

// GetRule serves a GET request for
// /api/projects/:project/rules/:id.
func (h *Handlers) GetRule(ctx *router.Context) {
	projectID, cfg, ok := obtainProjectConfigOrError(ctx)
	if !ok {
		return
	}
	ruleID := ctx.Params.ByName("id")
	if !rules.RuleIDRe.MatchString(ruleID) {
		http.Error(ctx.Writer, "Please supply a valid rule ID.", http.StatusBadRequest)
		return
	}
	txn, cancel := span.ReadOnlyTransaction(ctx.Context)
	defer cancel()

	r, err := rules.Read(txn, projectID, ruleID)
	if err != nil {
		if err == rules.NotExistsErr {
			http.Error(ctx.Writer, "Rule does not exist.", http.StatusNotFound)
			return
		}
		logging.Errorf(ctx.Context, "Reading rule %s: %s", ruleID, err)
		http.Error(ctx.Writer, "Internal server error.", http.StatusInternalServerError)
		return
	}

	respondWithRule(ctx, cfg, r)
}

// The rule returned by the REST API. This combines data stored in
// Spanner with output-only fields derived with the help of configuration.
type rule struct {
	rules.FailureAssociationRule

	BugLink *BugLink `json:"bugLink"`
}

func respondWithRule(ctx *router.Context, cfg *configpb.ProjectConfig, r *rules.FailureAssociationRule) {
	ctx.Writer.Header().Add("ETag", ruleETag(r))
	response := &rule{
		FailureAssociationRule: *r,
		BugLink:                createBugLink(r.BugID, cfg),
	}
	respondWithJSON(ctx, response)
}

func ruleETag(rule *rules.FailureAssociationRule) string {
	// While this ETag is strong, GAE's NGINX proxy will sometimes
	// remove them or modify them to be Weak as it compresses content.
	// Marking the ETag as weak to start with addresses this.
	return fmt.Sprintf(`W/"%s"`, rule.LastUpdated.UTC().Format(time.RFC3339Nano))
}

// Designed to conform to https://google.aip.dev/134.
type ruleUpdateRequest struct {
	Rule       *rules.FailureAssociationRule `json:"rule"`
	UpdateMask *field_mask.FieldMask         `json:"updateMask"`
	XSRFToken  string                        `json:"xsrfToken"`
}

// PatchRule serves a PATCH request for
// /api/projects/:project/rules/:id.
func (h *Handlers) PatchRule(ctx *router.Context) {
	projectID, cfg, ok := obtainProjectConfigOrError(ctx)
	if !ok {
		return
	}

	ruleID := ctx.Params.ByName("id")
	if !rules.RuleIDRe.MatchString(ruleID) {
		http.Error(ctx.Writer, "Please supply a valid rule ID.", http.StatusBadRequest)
		return
	}

	blob, err := ioutil.ReadAll(ctx.Request.Body)
	if err != nil {
		http.Error(ctx.Writer, "Failed to read request body.", http.StatusBadRequest)
		return
	}
	var request *ruleUpdateRequest
	if err := json.Unmarshal(blob, &request); err != nil {
		logging.Errorf(ctx.Context, "Failed to umarshal rule update request: %s", err)
		http.Error(ctx.Writer, "Incorrectly formed request: invalid json.", http.StatusBadRequest)
		return
	}

	if err := xsrf.Check(ctx.Context, request.XSRFToken); err != nil {
		http.Error(ctx.Writer, "Invalid XSRF Token.", http.StatusBadRequest)
		return
	}
	requestedETag := ctx.Request.Header.Get("If-Match")
	updatedRule, err := updateRule(ctx.Context, projectID, cfg, ruleID, request, requestedETag)
	if err != nil {
		if ConcurrentModificationTag.In(err) {
			http.Error(ctx.Writer, "The rule was modified since it was last read; the update was not applied.", http.StatusConflict)
			return
		}
		if ValidationErrorTag.In(err) {
			http.Error(ctx.Writer, fmt.Sprintf("Validation error: %s.", err.Error()), http.StatusBadRequest)
			return
		}
		logging.Errorf(ctx.Context, "Updating rule %s: %s", ruleID, err)
		http.Error(ctx.Writer, "Internal server error.", http.StatusInternalServerError)
		return
	}

	respondWithRule(ctx, cfg, updatedRule)
}

func updateRule(ctx context.Context, projectID string, cfg *configpb.ProjectConfig, ruleID string, request *ruleUpdateRequest, requestedETag string) (*rules.FailureAssociationRule, error) {
	user := auth.CurrentUser(ctx).Email

	var updatedRule *rules.FailureAssociationRule
	commitTime, err := span.ReadWriteTransaction(ctx, func(ctx context.Context) error {
		rule, err := rules.Read(ctx, projectID, ruleID)
		if err != nil {
			return err
		}
		if requestedETag != "" && ruleETag(rule) != requestedETag {
			return ConcurrentModificationTag.Apply(errors.New("the rule was modified since it was last read; the update was not applied."))
		}
		for _, path := range request.UpdateMask.Paths {
			// Only limited fields may be modified by the client.
			switch path {
			case "ruleDefinition":
				rule.RuleDefinition = request.Rule.RuleDefinition
			case "bugId":
				bugID := request.Rule.BugID
				if err := validateBugAgainstConfig(cfg, bugID); err != nil {
					return ValidationErrorTag.Apply(err)
				}

				// Verify the bug is not used by another rule.
				bugRule, err := rules.ReadByBug(ctx, bugID)
				if err != nil && err != rules.NotExistsErr {
					return err
				}
				bugValid := (err == rules.NotExistsErr) || (bugRule.Project == projectID && bugRule.RuleID == ruleID)
				if !bugValid {
					// Note: this validation could disclose the existence of rules
					// in projects other than those the user may have access to.
					// This is unavoidable in the context of the bug uniqueness
					// constraint we current have, which is needed to avoid Weetbix
					// making conflicting updates to the same bug.
					return ValidationErrorTag.Apply(fmt.Errorf("bug already used by another failure association rule (%s/%s)", bugRule.Project, bugRule.RuleID))
				}

				rule.BugID = bugID
			case "isActive":
				rule.IsActive = request.Rule.IsActive
			default:
				return ValidationErrorTag.Apply(fmt.Errorf("unsupported field mask: %s", path))
			}
		}

		if err := rules.Update(ctx, rule, user); err != nil {
			return ValidationErrorTag.Apply(err)
		}
		updatedRule = rule
		return nil
	})
	if err != nil {
		return nil, err
	}
	updatedRule.LastUpdated = commitTime.In(time.UTC)
	updatedRule.LastUpdatedUser = user
	return updatedRule, nil
}

// validateBugAgainstConfig validates the specified bug is consistent with
// the project configuration.
func validateBugAgainstConfig(cfg *configpb.ProjectConfig, bug bugs.BugID) error {
	switch bug.System {
	case bugs.MonorailSystem:
		project, _, err := bug.MonorailProjectAndID()
		if err != nil {
			return err
		}
		if project != cfg.Monorail.Project {
			return fmt.Errorf("bug not in expected monorail project (%s)", cfg.Monorail.Project)
		}
	default:
		return fmt.Errorf("unsupported bug system: %s", bug.System)
	}
	return nil
}
