// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package monorail

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/proto/mask"
	"google.golang.org/grpc"

	mpb "infra/monorailv2/api/v3/api_proto"
)

// projectsRE matches valid monorail project references.
var projectsRE = regexp.MustCompile(`projects/[a-z0-9\-_]+`)

// fakeIssuesClient provides a fake implementation of a monorail client, for testing. See:
// https://source.chromium.org/chromium/infra/infra/+/main:appengine/monorail/api/v3/api_proto/issues.proto
type fakeIssuesClient struct {
	store *FakeIssuesStore
	// User is the identity of the user interacting with monorail.
	user string
}

// UseFakeIssuesClient installs a given fake IssuesClient into the context so that
// it is used instead of making RPCs to monorail. The client will behave as if
// the given user is authenticated.
func UseFakeIssuesClient(ctx context.Context, store *FakeIssuesStore, user string) context.Context {
	f := &fakeIssuesClient{store: store, user: user}
	return context.WithValue(context.Background(), &testMonorailClientKey, mpb.IssuesClient(f))
}

func (f *fakeIssuesClient) GetIssue(ctx context.Context, in *mpb.GetIssueRequest, opts ...grpc.CallOption) (*mpb.Issue, error) {
	issue := f.issueByName(in.Name)
	if issue == nil {
		return nil, errors.New("issue not found")
	}
	// Copy proto so that if the consumer modifies the proto,
	// the stored proto does not change.
	return CopyIssue(issue.Issue), nil
}

func (f *fakeIssuesClient) issueByName(name string) *IssueData {
	for _, issue := range f.store.Issues {
		if issue.Issue.Name == name {
			return issue
		}
	}
	return nil
}

func (f *fakeIssuesClient) BatchGetIssues(ctx context.Context, in *mpb.BatchGetIssuesRequest, opts ...grpc.CallOption) (*mpb.BatchGetIssuesResponse, error) {
	result := &mpb.BatchGetIssuesResponse{}
	for _, name := range in.Names {
		issue := f.issueByName(name)
		if issue == nil {
			return nil, fmt.Errorf("issue %q not found", name)
		}
		// Copy proto so that if the consumer modifies the proto,
		// the stored proto does not change.
		result.Issues = append(result.Issues, CopyIssue(issue.Issue))
	}
	return result, nil
}

func (f *fakeIssuesClient) SearchIssues(ctx context.Context, in *mpb.SearchIssuesRequest, opts ...grpc.CallOption) (*mpb.SearchIssuesResponse, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeIssuesClient) ListComments(ctx context.Context, in *mpb.ListCommentsRequest, opts ...grpc.CallOption) (*mpb.ListCommentsResponse, error) {
	issue := f.issueByName(in.Parent)
	if issue == nil {
		return nil, fmt.Errorf("issue %q not found", in.Parent)
	}
	startIndex := 0
	if in.PageToken != "" {
		start, err := strconv.Atoi(in.PageToken)
		if err != nil {
			return nil, fmt.Errorf("invalid page token %q", in.PageToken)
		}
		startIndex = start
	}
	// The specification for ListComments says that 100 is both the default
	// and the maximum value.
	pageSize := int(in.PageSize)
	if pageSize > 100 || pageSize <= 0 {
		pageSize = 100
	}
	endIndex := startIndex + pageSize
	finished := false
	if endIndex > len(issue.Comments) {
		endIndex = len(issue.Comments)
		finished = true
	}
	comments := issue.Comments[startIndex:endIndex]

	result := &mpb.ListCommentsResponse{
		Comments: CopyComments(comments),
	}
	if !finished {
		result.NextPageToken = strconv.Itoa(endIndex)
	}
	return result, nil
}

// Implements a version of ModifyIssues that operates on local data instead of using monorail service.
// Reference implementation:
// https://source.chromium.org/chromium/infra/infra/+/main:appengine/monorail/api/v3/issues_servicer.py?q=%22def%20ModifyIssues%22
// https://source.chromium.org/chromium/infra/infra/+/main:appengine/monorail/api/v3/converters.py?q=IngestIssueDeltas&type=cs
func (f *fakeIssuesClient) ModifyIssues(ctx context.Context, in *mpb.ModifyIssuesRequest, opts ...grpc.CallOption) (*mpb.ModifyIssuesResponse, error) {
	// Current implementation would erroneously update the first issue
	// if the delta for the second issue failed validation. Currently our
	// fakes don't need this fidelity so it has not been implemented.
	if len(in.Deltas) > 1 {
		return nil, errors.New("not implemented for more than one delta")
	}
	var updatedIssues []*mpb.Issue
	for _, delta := range in.Deltas {
		name := delta.Issue.Name
		issue := f.issueByName(name)
		if issue == nil {
			return nil, fmt.Errorf("issue %q not found", name)
		}
		if !delta.UpdateMask.IsValid(issue.Issue) {
			return nil, fmt.Errorf("update mask for issue %q not valid", name)
		}
		const isFieldNameJSON = false
		const isUpdateMask = true
		m, err := mask.FromFieldMask(delta.UpdateMask, issue.Issue, isFieldNameJSON, isUpdateMask)
		if err != nil {
			return nil, errors.Annotate(err, "update mask for issue %q not valid", name).Err()
		}

		// Effect deletions.
		if len(delta.BlockedOnIssuesRemove) > 0 || len(delta.BlockingIssuesRemove) > 0 ||
			len(delta.CcsRemove) > 0 || len(delta.ComponentsRemove) > 0 || len(delta.FieldValsRemove) > 0 {
			return nil, errors.New("some removals are not supported by the current fake")
		}
		issue.Issue.Labels = mergeLabelDeletions(issue.Issue.Labels, delta.LabelsRemove)

		// Keep only the bits of the delta that are also in the field mask.
		filteredDelta := &mpb.Issue{}
		if err := m.Merge(delta.Issue, filteredDelta); err != nil {
			return nil, errors.Annotate(err, "failed to merge for issue %q", name).Err()
		}

		// Items in the delta's lists (like field values and labels) are treated as item-wise
		// additions or updates, not as a list-wise replacement.
		mergedDelta := CopyIssue(filteredDelta)
		mergedDelta.FieldValues = mergeFieldValues(issue.Issue.FieldValues, filteredDelta.FieldValues)
		mergedDelta.Labels = mergeLabels(issue.Issue.Labels, filteredDelta.Labels)
		if len(mergedDelta.BlockedOnIssueRefs) > 0 || len(mergedDelta.BlockingIssueRefs) > 0 ||
			len(mergedDelta.CcUsers) > 0 || len(mergedDelta.Components) > 0 {
			return nil, errors.New("some additions are not supported by the current fake")
		}

		// Apply the delta to the saved issue.
		if err := m.Merge(mergedDelta, issue.Issue); err != nil {
			return nil, errors.Annotate(err, "failed to merge for issue %q", name).Err()
		}

		// Currently only some amendments are created. Support for other
		// amendments can be added if needed.
		amendments := createAmendments(filteredDelta.Labels, delta.LabelsRemove, filteredDelta.FieldValues)
		issue.Comments = append(issue.Comments, &mpb.Comment{
			Name:       fmt.Sprintf("%s/comment/%v", name, len(issue.Comments)),
			State:      mpb.IssueContentState_ACTIVE,
			Type:       mpb.Comment_DESCRIPTION,
			Content:    in.CommentContent,
			Commenter:  f.user,
			Amendments: amendments,
		})
		// Copy the proto so that if the consumer modifies it, the saved proto
		// is not changed.
		updatedIssues = append(updatedIssues, CopyIssue(issue.Issue))
	}
	result := &mpb.ModifyIssuesResponse{
		Issues: updatedIssues,
	}
	return result, nil
}

func createAmendments(labelUpdate []*mpb.Issue_LabelValue, labelDeletions []string, fieldUpdates []*mpb.FieldValue) []*mpb.Comment_Amendment {
	var amendments []string
	for _, l := range labelUpdate {
		amendments = append(amendments, l.Label)
	}
	for _, l := range labelDeletions {
		amendments = append(amendments, "-"+l)
	}
	for _, fv := range fieldUpdates {
		if fv.Field == priorityFieldName {
			amendments = append(amendments, "Pri-"+fv.Value)
		}
		// Other field updates are currently not fully supported by the fake.
	}

	if len(amendments) > 0 {
		return []*mpb.Comment_Amendment{
			{
				FieldName:       "Labels",
				NewOrDeltaValue: strings.Join(amendments, " "),
			},
		}
	}
	return nil
}

// mergeFieldValues applies the updates in update to the existing field values
// and return the result.
func mergeFieldValues(existing []*mpb.FieldValue, update []*mpb.FieldValue) []*mpb.FieldValue {
	merge := make(map[string]*mpb.FieldValue)
	for _, fv := range existing {
		merge[fv.Field] = fv
	}
	for _, fv := range update {
		merge[fv.Field] = fv
	}
	var result []*mpb.FieldValue
	for _, v := range merge {
		result = append(result, v)
	}
	// Ensure the result of merging is predictable, as the order we iterate
	// over maps is not guaranteed.
	SortFieldValues(result)
	return result
}

// SortFieldValues sorts the given labels in alphabetical order.
func SortFieldValues(input []*mpb.FieldValue) {
	sort.Slice(input, func(i, j int) bool {
		return input[i].Field < input[j].Field
	})
}

// mergeLabels applies the updates in update to the existing labels
// and return the result.
func mergeLabels(existing []*mpb.Issue_LabelValue, update []*mpb.Issue_LabelValue) []*mpb.Issue_LabelValue {
	merge := make(map[string]*mpb.Issue_LabelValue)
	for _, l := range existing {
		merge[l.Label] = l
	}
	for _, l := range update {
		merge[l.Label] = l
	}
	var result []*mpb.Issue_LabelValue
	for _, v := range merge {
		result = append(result, v)
	}
	// Ensure the result of merging is predictable, as the order we iterate
	// over maps is not guaranteed.
	SortLabels(result)
	return result
}

func mergeLabelDeletions(existing []*mpb.Issue_LabelValue, deletes []string) []*mpb.Issue_LabelValue {
	merge := make(map[string]*mpb.Issue_LabelValue)
	for _, l := range existing {
		merge[l.Label] = l
	}
	for _, l := range deletes {
		delete(merge, l)
	}
	var result []*mpb.Issue_LabelValue
	for _, v := range merge {
		result = append(result, v)
	}
	// Ensure the result of merging is predictable, as the order we iterate
	// over maps is not guaranteed.
	SortLabels(result)
	return result
}

// SortLabels sorts the given labels in alphabetical order.
func SortLabels(input []*mpb.Issue_LabelValue) {
	sort.Slice(input, func(i, j int) bool {
		return input[i].Label < input[j].Label
	})
}

func (f *fakeIssuesClient) ModifyIssueApprovalValues(ctx context.Context, in *mpb.ModifyIssueApprovalValuesRequest, opts ...grpc.CallOption) (*mpb.ModifyIssueApprovalValuesResponse, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeIssuesClient) ListApprovalValues(ctx context.Context, in *mpb.ListApprovalValuesRequest, opts ...grpc.CallOption) (*mpb.ListApprovalValuesResponse, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeIssuesClient) ModifyCommentState(ctx context.Context, in *mpb.ModifyCommentStateRequest, opts ...grpc.CallOption) (*mpb.ModifyCommentStateResponse, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeIssuesClient) MakeIssueFromTemplate(ctx context.Context, in *mpb.MakeIssueFromTemplateRequest, opts ...grpc.CallOption) (*mpb.Issue, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeIssuesClient) MakeIssue(ctx context.Context, in *mpb.MakeIssueRequest, opts ...grpc.CallOption) (*mpb.Issue, error) {
	if !projectsRE.MatchString(in.Parent) {
		return nil, errors.New("parent project must be specified and match the form 'projects/{project_id}'")
	}
	// Copy the proto so that if the request proto is later modified, the save proto is not changed.
	saved := CopyIssue(in.Issue)
	saved.Name = fmt.Sprintf("%s/issues/%v", in.Parent, f.store.NextID)
	saved.Reporter = f.user

	f.store.NextID++
	issue := &IssueData{
		Issue: saved,
		Comments: []*mpb.Comment{
			{
				Name:      fmt.Sprintf("%s/comment/1", saved.Name),
				State:     mpb.IssueContentState_ACTIVE,
				Type:      mpb.Comment_DESCRIPTION,
				Content:   in.Description,
				Commenter: in.Issue.Reporter,
			},
		},
	}

	f.store.Issues = append(f.store.Issues, issue)

	// Copy the proto so that if the consumer modifies it, the saved proto is not changed.
	return CopyIssue(saved), nil
}
