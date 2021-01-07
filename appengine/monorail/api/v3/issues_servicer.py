# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import re

from api import resource_name_converters as rnc
from api.v3 import api_constants
from api.v3 import converters
from api.v3 import monorail_servicer
from api.v3 import paginator
from api.v3.api_proto import issues_pb2
from api.v3.api_proto import issue_objects_pb2
from api.v3.api_proto import issues_prpc_pb2
from businesslogic import work_env
from framework import exceptions

# We accept only the following filter, and only on ListComments.
# If we accept more complex filters in the future, introduce a library.
_APPROVAL_DEF_FILTER_RE = re.compile(
    r'approval = "(?P<approval_name>%s)"$' % rnc.APPROVAL_DEF_NAME_PATTERN)


class IssuesServicer(monorail_servicer.MonorailServicer):
  """Handle API requests related to Issue objects.
  Each API request is implemented with a method as defined in the
  .proto file that does any request-specific validation, uses work_env
  to safely operate on business objects, and returns a response proto.
  """

  DESCRIPTION = issues_prpc_pb2.IssuesServiceDescription

  @monorail_servicer.PRPCMethod
  def GetIssue(self, mc, request):
    # type: (MonorailContext, GetIssueRequest) -> Issue
    """pRPC API method that implements GetIssue.

    Raises:
      InputException: the given name does not have a valid format.
      NoSuchIssueException: the issue is not found.
      PermissionException the user is not allowed to view the issue.
    """
    issue_id = rnc.IngestIssueName(mc.cnxn, request.name, self.services)
    with work_env.WorkEnv(mc, self.services) as we:
      # TODO(crbug/monorail/7614): Eliminate the need to do this lookup.
      project = we.GetProjectByName(rnc.IngestProjectFromIssue(request.name))
      mc.LookupLoggedInUserPerms(project)
      issue = we.GetIssue(issue_id, allow_viewing_deleted=True)
    return self.converter.ConvertIssue(issue)

  @monorail_servicer.PRPCMethod
  def BatchGetIssues(self, mc, request):
    # type: (MonorailContext, BatchGetIssuesRequest) -> BatchGetIssuesResponse
    """pRPC API method that implements BatchGetIssues.

    Raises:
      InputException: If `names` is formatted incorrectly. Or if a parent
          collection in `names` does not match the value in `parent`.
      NoSuchIssueException: If any of the given issues do not exist.
      PermissionException If the requester does not have permission to view one
          (or more) of the given issues.
    """
    if len(request.names) > api_constants.MAX_BATCH_ISSUES:
      raise exceptions.InputException(
          'Requesting %d issues when the allowed maximum is %d issues.' %
          (len(request.names), api_constants.MAX_BATCH_ISSUES))
    if request.parent:
      parent_match = rnc._GetResourceNameMatch(
          request.parent, rnc.PROJECT_NAME_RE)
      parent_project = parent_match.group('project_name')
      with exceptions.ErrorAggregator(exceptions.InputException) as err_agg:
        for name in request.names:
          try:
            name_match = rnc._GetResourceNameMatch(name, rnc.ISSUE_NAME_RE)
            issue_project = name_match.group('project')
            if issue_project != parent_project:
              err_agg.AddErrorMessage(
                  '%s is not a child issue of %s.' % (name, request.parent))
          except exceptions.InputException as e:
            err_agg.AddErrorMessage(e.message)
    with work_env.WorkEnv(mc, self.services) as we:
      # NOTE(crbug/monorail/7614): Until the referenced cleanup is complete,
      # all servicer methods that are scoped to a single Project need to call
      # mc.LookupLoggedInUserPerms.
      #  This method does not because it may be scoped to multiple projects.
      issue_ids = rnc.IngestIssueNames(mc.cnxn, request.names, self.services)
      issues_by_iid = we.GetIssuesDict(issue_ids)
    return issues_pb2.BatchGetIssuesResponse(
        issues=self.converter.ConvertIssues(
            [issues_by_iid[issue_id] for issue_id in issue_ids]))

  @monorail_servicer.PRPCMethod
  def SearchIssues(self, mc, request):
    # type: (MonorailContext, SearchIssuesRequest) -> SearchIssuesResponse
    """pRPC API method that implements SearchIssue.

    Raises:
      InputException: if any given names in `projects` are invalid or if the
        search query uses invalid syntax (ie: unmatched parentheses).
    """
    page_size = paginator.CoercePageSize(
        request.page_size, api_constants.MAX_ISSUES_PER_PAGE)
    pager = paginator.Paginator(
        page_size=page_size,
        order_by=request.order_by,
        query=request.query,
        projects=request.projects)

    project_names = []
    for resource_name in request.projects:
      match = rnc._GetResourceNameMatch(resource_name, rnc.PROJECT_NAME_RE)
      project_names.append(match.group('project_name'))

    with work_env.WorkEnv(mc, self.services) as we:
      # NOTE(crbug/monorail/7614): Until the referenced cleanup is complete,
      # all servicer methods that are scoped to a single Project need to call
      # mc.LookupLoggedInUserPerms.
      #  This method does not because it may be scoped to multiple projects.
      list_result = we.SearchIssues(
          request.query, project_names, mc.auth.user_id, page_size,
          pager.GetStart(request.page_token), request.order_by)

    return issues_pb2.SearchIssuesResponse(
        issues=self.converter.ConvertIssues(list_result.items),
        next_page_token=pager.GenerateNextPageToken(list_result.next_start))

  @monorail_servicer.PRPCMethod
  def ListComments(self, mc, request):
    # type: (MonorailContext, ListCommentsRequest) -> ListCommentsResponse
    """pRPC API method that implements ListComments.

    Raises:
      InputException: the given name format or page_size are not valid.
      NoSuchIssueException: the parent is not found.
      PermissionException: the user is not allowed to view the parent.
    """
    issue_id = rnc.IngestIssueName(mc.cnxn, request.parent, self.services)
    page_size = paginator.CoercePageSize(
        request.page_size, api_constants.MAX_COMMENTS_PER_PAGE)
    pager = paginator.Paginator(
      parent=request.parent, page_size=page_size, filter_str=request.filter)
    approval_id = None
    if request.filter:
      match = _APPROVAL_DEF_FILTER_RE.match(request.filter)
      if match:
        approval_id = rnc.IngestApprovalDefName(
            mc.cnxn, match.group('approval_name'), self.services)
      if not match:
        raise exceptions.InputException(
            'Filtering other than approval not supported.')

    with work_env.WorkEnv(mc, self.services) as we:
      # TODO(crbug/monorail/7614): Eliminate the need to do this lookup.
      project = we.GetProjectByName(rnc.IngestProjectFromIssue(request.parent))
      mc.LookupLoggedInUserPerms(project)
      list_result = we.SafeListIssueComments(
          issue_id, page_size, pager.GetStart(request.page_token),
          approval_id=approval_id)
    return issues_pb2.ListCommentsResponse(
        comments=self.converter.ConvertComments(issue_id, list_result.items),
        next_page_token=pager.GenerateNextPageToken(list_result.next_start))

  @monorail_servicer.PRPCMethod
  def ListApprovalValues(self, mc, request):
    # type: (MonorailContext, ListApprovalValuesRequest) ->
    #     ListApprovalValuesResponse
    """pRPC API method that implements ListApprovalValues.

    Raises:
      InputException: the given parent does not have a valid format.
      NoSuchIssueException: the parent issue is not found.
      PermissionException the user is not allowed to view the parent issue.
    """
    issue_id = rnc.IngestIssueName(mc.cnxn, request.parent, self.services)
    with work_env.WorkEnv(mc, self.services) as we:
      # TODO(crbug/monorail/7614): Eliminate the need to do this lookup.
      project = we.GetProjectByName(rnc.IngestProjectFromIssue(request.parent))
      mc.LookupLoggedInUserPerms(project)
      issue = we.GetIssue(issue_id)

    api_avs = self.converter.ConvertApprovalValues(issue.approval_values,
        issue.field_values, issue.phases, issue_id=issue_id)

    return issues_pb2.ListApprovalValuesResponse(approval_values=api_avs)

  @monorail_servicer.PRPCMethod
  def MakeIssueFromTemplate(self, _mc, _request):
    # type: (MonorailContext, MakeIssueFromTemplateRequest) -> Issue
    """pRPC API method that implements MakeIssueFromTemplate.

    Raises:
      TODO(crbug/monorail/7197): Document errors when implemented
    """
    # Phase 1: Gather info
    #   Get project id and template name from template resource name.
    #   Get template pb.
    #   Make tracker_pb2.IssueDelta from request.template_issue_delta, share
    #   code with v3/ModifyIssue

    # with work_env.WorkEnv(mc, self.services) as we:
    #   project = ... get project from template.
    #   mc.LookupLoggedInUserPerms(project)
    #   created_issue = we.MakeIssueFromTemplate(template, description, delta)

    # Return newly created API issue.
    # return converters.ConvertIssue(created_issue)

    return issue_objects_pb2.Issue()

  @monorail_servicer.PRPCMethod
  def MakeIssue(self, mc, request):
    # type: (MonorailContext, MakeIssueRequest) -> Issue
    """pRPC API method that implements MakeIssue.

    Raises:
      InputException if any given names do not have a valid format or if any
        fields in the requested issue were invalid.
      NoSuchProjectException if no project exists with the given parent.
      FilterRuleException if proposed issue values violate any filter rules
        that shows error.
      PermissionException if user lacks sufficient permissions.
    """
    project_id = rnc.IngestProjectName(mc.cnxn, request.parent, self.services)
    with work_env.WorkEnv(mc, self.services) as we:
      # TODO(crbug/monorail/7614): Eliminate the need to do this lookup.
      project = we.GetProject(project_id)
      mc.LookupLoggedInUserPerms(project)

    ingested_issue = self.converter.IngestIssue(
        request.issue, project_id)
    send_email = self.converter.IngestNotifyType(request.notify_type)

    with work_env.WorkEnv(mc, self.services) as we:
      created_issue = we.MakeIssue(
          ingested_issue, request.description, send_email)
      starred_issue = we.StarIssue(created_issue, True)

    return self.converter.ConvertIssue(starred_issue)

  @monorail_servicer.PRPCMethod
  def ModifyIssues(self, mc, request):
    # type: (MonorailContext, ModifyIssuesRequest) -> ModifyIssuesResponse
    """pRPC API method that implements ModifyIssues.

    Raises:
      InputException if any given names do not have a valid format or if any
        fields in the requested issue were invalid.
      NoSuchIssueException if some issues weren't found.
      NoSuchProjectException if no project was found for some given issues.
      FilterRuleException if proposed issue changes violate any filter rules
        that shows error.
      PermissionException if user lacks sufficient permissions.
    """
    if not request.deltas:
      return issues_pb2.ModifyIssuesResponse()
    if len(request.deltas) > api_constants.MAX_MODIFY_ISSUES:
      raise exceptions.InputException(
          'Requesting %d updates when the allowed maximum is %d updates.' %
          (len(request.deltas), api_constants.MAX_MODIFY_ISSUES))
    impacted_issues_count = 0
    for delta in request.deltas:
      impacted_issues_count += (
          len(delta.blocked_on_issues_remove) +
          len(delta.blocking_issues_remove) +
          len(delta.issue.blocking_issue_refs) +
          len(delta.issue.blocked_on_issue_refs))
      if 'merged_into_issue_ref' in delta.update_mask.paths:
        impacted_issues_count += 1
    if impacted_issues_count > api_constants.MAX_MODIFY_IMPACTED_ISSUES:
      raise exceptions.InputException(
          'Updates include %d impacted issues when the allowed maximum is %d.' %
          (impacted_issues_count, api_constants.MAX_MODIFY_IMPACTED_ISSUES))
    iid_delta_pairs = self.converter.IngestIssueDeltas(request.deltas)
    with work_env.WorkEnv(mc, self.services) as we:
      issues = we.ModifyIssues(
          iid_delta_pairs,
          comment_content=request.comment_content,
          send_email=self.converter.IngestNotifyType(request.notify_type))

    return issues_pb2.ModifyIssuesResponse(
        issues=self.converter.ConvertIssues(issues))

  @monorail_servicer.PRPCMethod
  def ModifyIssueApprovalValues(self, mc, request):
    # type: (MonorailContext, ModifyIssueApprovalValuesRequest) ->
    #     ModifyIssueApprovalValuesResponse
    """pRPC API method that implements ModifyIssueApprovalValues.

    Raises:
      InputException if any fields in the delta were invalid.
      NoSuchIssueException: if the issue of any ApprovalValue isn't found.
      NoSuchProjectException: if the parent project of any ApprovalValue isn't
          found.
      NoSuchUserException: if any user value provided isn't found.
      PermissionException if user lacks sufficient permissions.
      # TODO(crbug/monorail/7925): Not all of these are yet thrown.
    """
    if len(request.deltas) > api_constants.MAX_MODIFY_APPROVAL_VALUES:
      raise exceptions.InputException(
          'Requesting %d updates when the allowed maximum is %d updates.' %
          (len(request.deltas), api_constants.MAX_MODIFY_APPROVAL_VALUES))
    response = issues_pb2.ModifyIssueApprovalValuesResponse()
    delta_specifications = self.converter.IngestApprovalDeltas(
        request.deltas, mc.auth.user_id)
    send_email = self.converter.IngestNotifyType(request.notify_type)
    with work_env.WorkEnv(mc, self.services) as we:
      # NOTE(crbug/monorail/7614): Until the referenced cleanup is complete,
      # all servicer methods that are scoped to a single Project need to call
      # mc.LookupLoggedInUserPerms.
      # This method does not because it may be scoped to multiple projects.
      issue_approval_values = we.BulkUpdateIssueApprovalsV3(
          delta_specifications, request.comment_content, send_email=send_email)
    api_avs = []
    for issue, approval_value in issue_approval_values:
      api_avs.extend(
          self.converter.ConvertApprovalValues(
              [approval_value],
              issue.field_values,
              issue.phases,
              issue_id=issue.issue_id))
    response.approval_values.extend(api_avs)
    return response

  @monorail_servicer.PRPCMethod
  def ModifyCommentState(self, mc, request):
    # type: (MonorailContext, ModifyCommentStateRequest) ->
    #     ModifyCommentStateResponse
    """pRPC API method that implements ModifyCommentState.

    We do not support changing between DELETED <-> SPAM. User must
    undelete or unflag-as-spam first.

    Raises:
      NoSuchProjectException if the parent Project does not exist.
      NoSuchIssueException: if the issue does not exist.
      NoSuchCommentException: if the comment does not exist.
      PermissionException if user lacks sufficient permissions.
      ActionNotSupported if user requests unsupported state transitions.
    """
    (project_id, issue_id,
     comment_num) = rnc.IngestCommentName(mc.cnxn, request.name, self.services)
    with work_env.WorkEnv(mc, self.services) as we:
      # TODO(crbug/monorail/7614): Eliminate the need to do this lookup.
      project = we.GetProject(project_id)
      mc.LookupLoggedInUserPerms(project)
      issue = we.GetIssue(issue_id, use_cache=False)
      comments_list = we.SafeListIssueComments(issue_id, 1, comment_num).items
      try:
        comment = comments_list[0]
      except IndexError:
        raise exceptions.NoSuchCommentException()

      if request.state == issue_objects_pb2.IssueContentState.Value('ACTIVE'):
        if comment.is_spam:
          we.FlagComment(issue, comment, False)
        elif comment.deleted_by != 0:
          we.DeleteComment(issue, comment, delete=False)
        else:
          # No-op if already currently active
          pass
      elif request.state == issue_objects_pb2.IssueContentState.Value(
          'DELETED'):
        if (not comment.deleted_by) and (not comment.is_spam):
          we.DeleteComment(issue, comment, delete=True)
        elif comment.deleted_by and not comment.is_spam:
          # No-op if already deleted
          pass
        else:
          raise exceptions.ActionNotSupported(
              'Cannot change comment state from spam to deleted.')
      elif request.state == issue_objects_pb2.IssueContentState.Value('SPAM'):
        if (not comment.deleted_by) and (not comment.is_spam):
          we.FlagComment(issue, comment, True)
        elif comment.is_spam:
          # No-op if already spam
          pass
        else:
          raise exceptions.ActionNotSupported(
              'Cannot change comment state from deleted to spam.')
      else:
        raise exceptions.ActionNotSupported('Unsupported target comment state.')

      # FlagComment does not have side effect on comment, must refresh.
      refreshed_comment = we.SafeListIssueComments(issue_id, 1,
                                                   comment_num).items[0]

    converted_comment = self.converter.ConvertComments(
        issue_id, [refreshed_comment])[0]
    return issues_pb2.ModifyCommentStateResponse(comment=converted_comment)
