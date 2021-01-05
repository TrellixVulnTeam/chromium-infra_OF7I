# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import collections
import itertools
import logging
import time

from google.protobuf import timestamp_pb2

from api import resource_name_converters as rnc
from api.v3.api_proto import feature_objects_pb2
from api.v3.api_proto import issues_pb2
from api.v3.api_proto import issue_objects_pb2
from api.v3.api_proto import project_objects_pb2
from api.v3.api_proto import user_objects_pb2

from framework import exceptions
from framework import framework_bizobj
from framework import framework_constants
from framework import framework_helpers
from proto import tracker_pb2
from project import project_helpers
from tracker import attachment_helpers
from tracker import field_helpers
from tracker import tracker_bizobj as tbo
from tracker import tracker_helpers

Choice = project_objects_pb2.FieldDef.EnumTypeSettings.Choice

# Ingest/convert dicts for ApprovalStatus.
_V3_APPROVAL_STATUS = issue_objects_pb2.ApprovalValue.ApprovalStatus.Value
_APPROVAL_STATUS_INGEST = {
  _V3_APPROVAL_STATUS('APPROVAL_STATUS_UNSPECIFIED'): None,
  _V3_APPROVAL_STATUS('NOT_SET'): tracker_pb2.ApprovalStatus.NOT_SET,
  _V3_APPROVAL_STATUS('NEEDS_REVIEW'): tracker_pb2.ApprovalStatus.NEEDS_REVIEW,
  _V3_APPROVAL_STATUS('NA'): tracker_pb2.ApprovalStatus.NA,
  _V3_APPROVAL_STATUS('REVIEW_REQUESTED'):
      tracker_pb2.ApprovalStatus.REVIEW_REQUESTED,
  _V3_APPROVAL_STATUS('REVIEW_STARTED'):
      tracker_pb2.ApprovalStatus.REVIEW_STARTED,
  _V3_APPROVAL_STATUS('NEED_INFO'): tracker_pb2.ApprovalStatus.NEED_INFO,
  _V3_APPROVAL_STATUS('APPROVED'): tracker_pb2.ApprovalStatus.APPROVED,
  _V3_APPROVAL_STATUS('NOT_APPROVED'): tracker_pb2.ApprovalStatus.NOT_APPROVED,
}
_APPROVAL_STATUS_CONVERT = {
  val: key for key, val in _APPROVAL_STATUS_INGEST.items()}


class Converter(object):
  """Class to manage converting objects between the API and backend layer."""

  def __init__(self, mc, services):
    # type: (MonorailContext, Services) -> Converter
    """Create a Converter with the given MonorailContext and Services.

    Args:
      mc: MonorailContext object containing the MonorailConnection to the DB
            and the requester's AuthData object.
      services: Services object for connections to backend services.
    """
    self.cnxn = mc.cnxn
    self.user_auth = mc.auth
    self.services = services

  # Hotlists

  def ConvertHotlist(self, hotlist):
    # type: (proto.feature_objects_pb2.Hotlist)
    #    -> api_proto.feature_objects_pb2.Hotlist
    """Convert a protorpc Hotlist into a protoc Hotlist."""

    hotlist_resource_name = rnc.ConvertHotlistName(hotlist.hotlist_id)
    members_by_id = rnc.ConvertUserNames(
        hotlist.owner_ids + hotlist.editor_ids)
    default_columns = self._ComputeIssuesListColumns(hotlist.default_col_spec)
    if hotlist.is_private:
      hotlist_privacy = feature_objects_pb2.Hotlist.HotlistPrivacy.Value(
          'PRIVATE')
    else:
      hotlist_privacy = feature_objects_pb2.Hotlist.HotlistPrivacy.Value(
          'PUBLIC')

    return feature_objects_pb2.Hotlist(
        name=hotlist_resource_name,
        display_name=hotlist.name,
        owner=members_by_id.get(hotlist.owner_ids[0]),
        editors=[
            members_by_id.get(editor_id) for editor_id in hotlist.editor_ids
        ],
        summary=hotlist.summary,
        description=hotlist.description,
        default_columns=default_columns,
        hotlist_privacy=hotlist_privacy)

  def ConvertHotlists(self, hotlists):
    # type: (Sequence[proto.feature_objects_pb2.Hotlist])
    #    -> Sequence[api_proto.feature_objects_pb2.Hotlist]
    """Convert protorpc Hotlists into protoc Hotlists."""
    return [self.ConvertHotlist(hotlist) for hotlist in hotlists]

  def ConvertHotlistItems(self, hotlist_id, items):
    # type: (int, Sequence[proto.features_pb2.HotlistItem]) ->
    #     Sequence[api_proto.feature_objects_pb2.Hotlist]
    """Convert a Sequence of protorpc HotlistItems into a Sequence of protoc
       HotlistItems.

    Args:
      hotlist_id: ID of the Hotlist the items belong to.
      items: Sequence of HotlistItem protorpc objects.

    Returns:
      Sequence of protoc HotlistItems in the same order they are given in
        `items`.
      In the rare event that any issues in `items` are not found, they will be
        omitted from the result.
    """
    issue_ids = [item.issue_id for item in items]
    # Converting HotlistItemNames and IssueNames both require looking up the
    # issues in the hotlist. However, we want to keep the code clean and
    # readable so we keep the two processes separate.
    resource_names_dict = rnc.ConvertHotlistItemNames(
        self.cnxn, hotlist_id, issue_ids, self.services)
    issue_names_dict = rnc.ConvertIssueNames(
        self.cnxn, issue_ids, self.services)
    adders_by_id = rnc.ConvertUserNames([item.adder_id for item in items])

    # Filter out items whose issues were not found.
    found_items = [
        item for item in items if resource_names_dict.get(item.issue_id) and
        issue_names_dict.get(item.issue_id)
    ]
    if len(items) != len(found_items):
      found_ids = [item.issue_id for item in found_items]
      missing_ids = [iid for iid in issue_ids if iid not in found_ids]
      logging.info('HotlistItem issues %r not found' % missing_ids)

    # Generate user friendly ranks (0, 1, 2, 3,...) that are exposed to API
    # clients, instead of using padded ranks (1, 11, 21, 31,...).
    sorted_ranks = sorted(item.rank for item in found_items)
    friendly_ranks_dict = {
        rank: friendly_rank for friendly_rank, rank in enumerate(sorted_ranks)
    }

    api_items = []
    for item in found_items:
      api_item = feature_objects_pb2.HotlistItem(
          name=resource_names_dict.get(item.issue_id),
          issue=issue_names_dict.get(item.issue_id),
          rank=friendly_ranks_dict[item.rank],
          adder=adders_by_id.get(item.adder_id),
          note=item.note)
      if item.date_added:
        api_item.create_time.FromSeconds(item.date_added)
      api_items.append(api_item)

    return api_items

  # Issues

  def _ConvertComponentValues(self, issue):
    # proto.tracker_pb2.Issue ->
    #     Sequence[api_proto.issue_objects_pb2.Issue.ComponentValue]
    """Convert the status string on issue into a ComponentValue."""
    component_values = []
    component_ids = itertools.chain(
        issue.component_ids, issue.derived_component_ids)
    ids_to_names = rnc.ConvertComponentDefNames(
        self.cnxn, component_ids, issue.project_id, self.services)

    for component_id in issue.component_ids:
      if component_id in ids_to_names:
        component_values.append(
            issue_objects_pb2.Issue.ComponentValue(
                component=ids_to_names[component_id],
                derivation=issue_objects_pb2.Derivation.Value(
                    'EXPLICIT')))
    for derived_component_id in issue.derived_component_ids:
      if derived_component_id in ids_to_names:
        component_values.append(
            issue_objects_pb2.Issue.ComponentValue(
                component=ids_to_names[derived_component_id],
                derivation=issue_objects_pb2.Derivation.Value('RULE')))

    return component_values

  def _ConvertStatusValue(self, issue):
    # proto.tracker_pb2.Issue -> api_proto.issue_objects_pb2.Issue.StatusValue
    """Convert the status string on issue into a StatusValue."""
    derivation = issue_objects_pb2.Derivation.Value(
        'DERIVATION_UNSPECIFIED')
    if issue.status:
      derivation = issue_objects_pb2.Derivation.Value('EXPLICIT')
    else:
      derivation = issue_objects_pb2.Derivation.Value('RULE')
    return issue_objects_pb2.Issue.StatusValue(
        status=issue.status or issue.derived_status, derivation=derivation)

  def _ConvertAmendments(self, amendments, user_display_names):
    # type: (Sequence[proto.tracker_pb2.Amendment], Mapping[int, str]) ->
    #     Sequence[api_proto.issue_objects_pb2.Comment.Amendment]
    """Convert protorpc Amendments to protoc Amendments.

    Args:
      amendments: the amendments to convert
      user_display_names: map from user_id to display name for all users
          involved in the amendments.

    Returns:
      The converted amendments.
    """
    results = []
    for amendment in amendments:
      field_name = tbo.GetAmendmentFieldName(amendment)
      new_value = tbo.AmendmentString_New(amendment, user_display_names)
      results.append(
          issue_objects_pb2.Comment.Amendment(
              field_name=field_name,
              new_or_delta_value=new_value,
              old_value=amendment.oldvalue))
    return results

  def _ConvertAttachments(self, attachments, project_name):
    # type: (Sequence[proto.tracker_pb2.Attachment], str) ->
    #     Sequence[api_proto.issue_objects_pb2.Comment.Attachment]
    """Convert protorpc Attachments to protoc Attachments."""
    results = []
    for attach in attachments:
      if attach.deleted:
        state = issue_objects_pb2.IssueContentState.Value('DELETED')
        size, thumbnail_uri, view_uri, download_uri = None, None, None, None
      else:
        state = issue_objects_pb2.IssueContentState.Value('ACTIVE')
        size = attach.filesize
        download_uri = attachment_helpers.GetDownloadURL(attach.attachment_id)
        view_uri = attachment_helpers.GetViewURL(
            attach, download_uri, project_name)
        thumbnail_uri = attachment_helpers.GetThumbnailURL(attach, download_uri)
      results.append(
          issue_objects_pb2.Comment.Attachment(
              filename=attach.filename,
              state=state,
              size=size,
              media_type=attach.mimetype,
              thumbnail_uri=thumbnail_uri,
              view_uri=view_uri,
              download_uri=download_uri))
    return results

  def ConvertComments(self, issue_id, comments):
    # type: (int, Sequence[proto.tracker_pb2.IssueComment])
    #     -> Sequence[api_proto.issue_objects_pb2.Comment]
    """Convert protorpc IssueComments from issue into protoc Comments."""
    issue = self.services.issue.GetIssue(self.cnxn, issue_id)
    users_by_id = self.services.user.GetUsersByIDs(
        self.cnxn, tbo.UsersInvolvedInCommentList(comments))
    (user_display_names,
     _user_display_emails) = framework_bizobj.CreateUserDisplayNamesAndEmails(
         self.cnxn, self.services, self.user_auth, users_by_id.values())
    comment_names_dict = rnc.CreateCommentNames(
        issue.local_id, issue.project_name,
        [comment.sequence for comment in comments])
    approval_ids = [
        comment.approval_id
        for comment in comments
        if comment.approval_id is not None  # In case of a 0 approval_id.
    ]
    approval_ids_to_names = rnc.ConvertApprovalDefNames(
        self.cnxn, approval_ids, issue.project_id, self.services)

    converted_comments = []
    for comment in comments:
      if comment.is_spam:
        state = issue_objects_pb2.IssueContentState.Value('SPAM')
      elif comment.deleted_by:
        state = issue_objects_pb2.IssueContentState.Value('DELETED')
      else:
        state = issue_objects_pb2.IssueContentState.Value('ACTIVE')
      comment_type = issue_objects_pb2.Comment.Type.Value('COMMENT')
      if comment.is_description:
        comment_type = issue_objects_pb2.Comment.Type.Value('DESCRIPTION')
      converted_attachments = self._ConvertAttachments(
          comment.attachments, issue.project_name)
      converted_amendments = self._ConvertAmendments(
          comment.amendments, user_display_names)
      converted_comment = issue_objects_pb2.Comment(
          name=comment_names_dict[comment.sequence],
          state=state,
          type=comment_type,
          create_time=timestamp_pb2.Timestamp(seconds=comment.timestamp),
          attachments=converted_attachments,
          amendments=converted_amendments)
      if comment.content:
        converted_comment.content = comment.content
      if comment.user_id:
        converted_comment.commenter = rnc.ConvertUserName(comment.user_id)
      if comment.inbound_message:
        converted_comment.inbound_message = comment.inbound_message
      if comment.approval_id and comment.approval_id in approval_ids_to_names:
        converted_comment.approval = approval_ids_to_names[comment.approval_id]
      converted_comments.append(converted_comment)
    return converted_comments

  def ConvertIssue(self, issue):
    # type: (proto.tracker_pb2.Issue) -> api_proto.issue_objects_pb2.Issue
    """Convert a protorpc Issue into a protoc Issue."""
    issues = self.ConvertIssues([issue])
    if len(issues) < 1:
      raise exceptions.NoSuchIssueException()
    if len(issues) > 1:
      logging.warning('More than one converted issue returned: %s', issues)
    return issues[0]

  def ConvertIssues(self, issues):
    # type: (Sequence[proto.tracker_pb2.Issue]) ->
    #     Sequence[api_proto.issue_objects_pb2.Issue]
    """Convert protorpc Issues into protoc Issues."""
    issue_ids = [issue.issue_id for issue in issues]
    issue_names_dict = rnc.ConvertIssueNames(
        self.cnxn, issue_ids, self.services)
    found_issues = [
        issue for issue in issues if issue.issue_id in issue_names_dict
    ]
    converted_issues = []
    for issue in found_issues:
      status = self._ConvertStatusValue(issue)
      content_state = issue_objects_pb2.IssueContentState.Value(
          'STATE_UNSPECIFIED')
      if issue.is_spam:
        content_state = issue_objects_pb2.IssueContentState.Value('SPAM')
      elif issue.deleted:
        content_state = issue_objects_pb2.IssueContentState.Value('DELETED')
      else:
        content_state = issue_objects_pb2.IssueContentState.Value('ACTIVE')

      owner = None
      # Explicit values override values derived from rules.
      if issue.owner_id:
        owner = issue_objects_pb2.Issue.UserValue(
            derivation=issue_objects_pb2.Derivation.Value('EXPLICIT'),
            user=rnc.ConvertUserName(issue.owner_id))
      elif issue.derived_owner_id:
        owner = issue_objects_pb2.Issue.UserValue(
            derivation=issue_objects_pb2.Derivation.Value('RULE'),
            user=rnc.ConvertUserName(issue.derived_owner_id))

      cc_users = []
      for cc_user_id in issue.cc_ids:
        cc_users.append(
            issue_objects_pb2.Issue.UserValue(
                derivation=issue_objects_pb2.Derivation.Value('EXPLICIT'),
                user=rnc.ConvertUserName(cc_user_id)))
      for derived_cc_user_id in issue.derived_cc_ids:
        cc_users.append(
            issue_objects_pb2.Issue.UserValue(
                derivation=issue_objects_pb2.Derivation.Value('RULE'),
                user=rnc.ConvertUserName(derived_cc_user_id)))

      labels = self.ConvertLabels(
          issue.labels, issue.derived_labels, issue.project_id)
      components = self._ConvertComponentValues(issue)
      non_approval_fvs = self._GetNonApprovalFieldValues(
          issue.field_values, issue.project_id)
      field_values = self.ConvertFieldValues(
          non_approval_fvs, issue.project_id, issue.phases)
      field_values.extend(
          self.ConvertEnumFieldValues(
              issue.labels, issue.derived_labels, issue.project_id))
      related_issue_ids = (
          [issue.merged_into] + issue.blocked_on_iids + issue.blocking_iids)
      issue_names_by_ids = rnc.ConvertIssueNames(
          self.cnxn, related_issue_ids, self.services)
      merged_into_issue_ref = None
      if issue.merged_into and issue.merged_into in issue_names_by_ids:
        merged_into_issue_ref = issue_objects_pb2.IssueRef(
            issue=issue_names_by_ids[issue.merged_into])
      if issue.merged_into_external:
        merged_into_issue_ref = issue_objects_pb2.IssueRef(
            ext_identifier=issue.merged_into_external)

      blocked_on_issue_refs = [
          issue_objects_pb2.IssueRef(issue=issue_names_by_ids[iid])
          for iid in issue.blocked_on_iids
          if iid in issue_names_by_ids
      ]
      blocked_on_issue_refs.extend(
          issue_objects_pb2.IssueRef(
              ext_identifier=blocked_on.ext_issue_identifier)
          for blocked_on in issue.dangling_blocked_on_refs)

      blocking_issue_refs = [
          issue_objects_pb2.IssueRef(issue=issue_names_by_ids[iid])
          for iid in issue.blocking_iids
          if iid in issue_names_by_ids
      ]
      blocking_issue_refs.extend(
          issue_objects_pb2.IssueRef(
              ext_identifier=blocking.ext_issue_identifier)
          for blocking in issue.dangling_blocking_refs)
      # All other timestamps were set when the issue was created.
      close_time = None
      if issue.closed_timestamp:
        close_time = timestamp_pb2.Timestamp(seconds=issue.closed_timestamp)

      phases = self._ComputePhases(issue.phases)

      result = issue_objects_pb2.Issue(
          name=issue_names_dict[issue.issue_id],
          summary=issue.summary,
          state=content_state,
          status=status,
          reporter=rnc.ConvertUserName(issue.reporter_id),
          owner=owner,
          cc_users=cc_users,
          labels=labels,
          components=components,
          field_values=field_values,
          merged_into_issue_ref=merged_into_issue_ref,
          blocked_on_issue_refs=blocked_on_issue_refs,
          blocking_issue_refs=blocking_issue_refs,
          create_time=timestamp_pb2.Timestamp(seconds=issue.opened_timestamp),
          close_time=close_time,
          modify_time=timestamp_pb2.Timestamp(seconds=issue.modified_timestamp),
          component_modify_time=timestamp_pb2.Timestamp(
              seconds=issue.component_modified_timestamp),
          status_modify_time=timestamp_pb2.Timestamp(
              seconds=issue.status_modified_timestamp),
          owner_modify_time=timestamp_pb2.Timestamp(
              seconds=issue.owner_modified_timestamp),
          star_count=issue.star_count,
          phases=phases)
      # TODO(crbug.com/monorail/5857): Set attachment_count unconditionally
      # after the underlying source of negative attachment counts has been
      # resolved and database has been repaired.
      if issue.attachment_count >= 0:
        result.attachment_count = issue.attachment_count
      converted_issues.append(result)
    return converted_issues

  def IngestIssueDeltas(self, issue_deltas):
    # type: (Sequence[api_proto.issues_pb2.IssueDelta]) ->
    #     Sequence[Tuple[int, proto.tracker_pb2.IssueDelta]]
    """Ingests protoc IssueDeltas, into protorpc IssueDeltas.

    Args:
      issue_deltas: the protoc IssueDeltas to ingest.

    Returns:
      A list of (issue_id, tracker_pb2.IssueDelta) tuples that contain
      values found in issue_deltas, ignoring all OUTPUT_ONLY and masked
      fields.

    Raises:
      InputException: if any fields in the approval_deltas were invalid.
      NoSuchProjectException: if any parent projects are not found.
      NoSuchIssueException: if any issues are not found.
      NoSuchComponentException: if any components are not found.
    """
    issue_names = [delta.issue.name for delta in issue_deltas]
    issue_ids = rnc.IngestIssueNames(self.cnxn, issue_names, self.services)
    issues_dict, misses = self.services.issue.GetIssuesDict(
        self.cnxn, issue_ids)
    if misses:
      logging.info(
          'Issues not found for supposedly valid issue_ids: %r' % misses)
      raise ValueError('Could not fetch some issues.')
    configs_by_pid = self.services.config.GetProjectConfigs(
        self.cnxn, {issue.project_id for issue in issues_dict.values()})

    with exceptions.ErrorAggregator(exceptions.InputException) as err_agg:
      for api_delta in issue_deltas:
        if not api_delta.HasField('update_mask'):
          err_agg.AddErrorMessage(
              '`update_mask` must be set for {} delta.', api_delta.issue.name)
        elif not api_delta.update_mask.IsValidForDescriptor(
            issue_objects_pb2.Issue.DESCRIPTOR):
          err_agg.AddErrorMessage(
              'Invalid `update_mask` for {} delta.', api_delta.issue.name)

    ingested = []
    for iid, api_delta in zip(issue_ids, issue_deltas):
      delta = tracker_pb2.IssueDelta()

      # Check non-repeated fields before MergeMessage because in an object
      # where fields are not set and with a FieldMask applied, there is no
      # way to tell if empty fields were explicitly listed or not listed
      # in the FieldMask.
      paths_set = set(api_delta.update_mask.paths)
      if (not paths_set.isdisjoint({'status', 'status.status'}) and
          api_delta.issue.status.status):
        delta.status = api_delta.issue.status.status
      elif 'status.status' in paths_set and not api_delta.issue.status.status:
        delta.status = ''

      if (not paths_set.isdisjoint({'owner', 'owner.user'}) and
          api_delta.issue.owner.user):
        delta.owner_id = rnc.IngestUserName(
              self.cnxn, api_delta.issue.owner.user, self.services)
      elif 'owner.user' in paths_set and not api_delta.issue.owner.user:
        delta.owner_id = framework_constants.NO_USER_SPECIFIED

      if 'summary' in paths_set:
        if api_delta.issue.summary:
          delta.summary = api_delta.issue.summary
        else:
          delta.summary = ''

      merge_ref = api_delta.issue.merged_into_issue_ref
      if 'merged_into_issue_ref' in paths_set:
        if (api_delta.issue.merged_into_issue_ref.issue or
            api_delta.issue.merged_into_issue_ref.ext_identifier):
          ingested_ref = self._IngestIssueRef(merge_ref)
          if isinstance(ingested_ref, tracker_pb2.DanglingIssueRef):
            delta.merged_into_external = ingested_ref.ext_issue_identifier
          else:
            delta.merged_into = ingested_ref
      elif 'merged_into_issue_ref.issue' in paths_set:
        if api_delta.issue.merged_into_issue_ref.issue:
          delta.merged_into = self._IngestIssueRef(merge_ref)
        else:
          delta.merged_into = 0
      elif 'merged_into_issue_ref.ext_identifier' in paths_set:
        if api_delta.issue.merged_into_issue_ref.ext_identifier:
          ingested_ref = self._IngestIssueRef(merge_ref)
          delta.merged_into_external = ingested_ref.ext_issue_identifier
        else:
          delta.merged_into_external = ''

      filtered_api_issue = issue_objects_pb2.Issue()
      api_delta.update_mask.MergeMessage(
          api_delta.issue,
          filtered_api_issue,
          replace_message_field=True,
          replace_repeated_field=True)

      cc_names = [name for name in api_delta.ccs_remove] + [
          user_value.user for user_value in filtered_api_issue.cc_users
      ]
      cc_ids = rnc.IngestUserNames(self.cnxn, cc_names, self.services)
      delta.cc_ids_remove = cc_ids[:len(api_delta.ccs_remove)]
      delta.cc_ids_add = cc_ids[len(api_delta.ccs_remove):]

      comp_names = [component for component in api_delta.components_remove] + [
          c_value.component for c_value in filtered_api_issue.components
      ]
      project_comp_ids = rnc.IngestComponentDefNames(
          self.cnxn, comp_names, self.services)
      comp_ids = [comp_id for (_pid, comp_id) in project_comp_ids]
      delta.comp_ids_remove = comp_ids[:len(api_delta.components_remove)]
      delta.comp_ids_add = comp_ids[len(api_delta.components_remove):]

      # Added to delta below, after ShiftEnumFieldsIntoLabels.
      labels_add = [value.label for value in filtered_api_issue.labels]
      labels_remove = [label for label in api_delta.labels_remove]

      config = configs_by_pid[issues_dict[iid].project_id]
      fvs_add, add_enums = self._IngestFieldValues(
          filtered_api_issue.field_values, config)
      fvs_remove, remove_enums = self._IngestFieldValues(
          api_delta.field_vals_remove, config)
      field_helpers.ShiftEnumFieldsIntoLabels(
          labels_add, labels_remove, add_enums, remove_enums, config)
      delta.field_vals_add = fvs_add
      delta.field_vals_remove = fvs_remove
      delta.labels_add = labels_add
      delta.labels_remove = labels_remove
      assert len(add_enums) == 0  # ShiftEnumFieldsIntoLabels clears all enums.
      assert len(remove_enums) == 0

      blocked_on_iids_rm, blocked_on_dangling_rm = self._IngestIssueRefs(
          api_delta.blocked_on_issues_remove)
      delta.blocked_on_remove = blocked_on_iids_rm
      delta.ext_blocked_on_remove = [
          ref.ext_issue_identifier for ref in blocked_on_dangling_rm
      ]

      blocked_on_iids_add, blocked_on_dangling_add = self._IngestIssueRefs(
          filtered_api_issue.blocked_on_issue_refs)
      delta.blocked_on_add = blocked_on_iids_add
      delta.ext_blocked_on_add = [
          ref.ext_issue_identifier for ref in blocked_on_dangling_add
      ]

      blocking_iids_rm, blocking_dangling_rm = self._IngestIssueRefs(
          api_delta.blocking_issues_remove)
      delta.blocking_remove = blocking_iids_rm
      delta.ext_blocking_remove = [
          ref.ext_issue_identifier for ref in blocking_dangling_rm
      ]

      blocking_iids_add, blocking_dangling_add = self._IngestIssueRefs(
          filtered_api_issue.blocking_issue_refs)
      delta.blocking_add = blocking_iids_add
      delta.ext_blocking_add = [
          ref.ext_issue_identifier for ref in blocking_dangling_add
      ]

      ingested.append((iid, delta))

    return ingested

  def IngestApprovalDeltas(self, approval_deltas, setter_id):
    # type: (Sequence[api_proto.issues_pb2.ApprovalDelta], int) ->
    #     Sequence[Tuple[int, int, proto.tracker_pb2.ApprovalDelta]]
    """Ingests protoc ApprovalDeltas into protorpc ApprovalDeltas.

    Args:
      approval_deltas: the protoc ApprovalDeltas to ingest.
      setter_id: The ID for the user setting the deltas.

    Returns:
      Sequence of (issue_id, approval_id, ApprovalDelta) tuples in the order
      provided. The ApprovalDeltas ignore all OUTPUT_ONLY and masked fields.
      The tuples are "delta_specifications;" they identify one requested change.

    Raises:
      InputException: if any fields in the approval_delta protos were invalid.
      NoSuchProjectException: if the parent project of any ApprovalValue isn't
          found.
      NoSuchIssueException: if the issue of any ApprovalValue isn't found.
      NoSuchUserException: if any user value was provided with an invalid email.
          Note that users specified by ID are not checked for existence.
    """
    delta_specifications = []
    set_on = int(time.time())  # Use the same timestamp for all deltas.
    for approval_delta in approval_deltas:
      approval_name = approval_delta.approval_value.name
      # TODO(crbug/monorail/8173): Aggregate errors.
      project_id, issue_id, approval_id = rnc.IngestApprovalValueName(
          self.cnxn, approval_name, self.services)

      if not approval_delta.HasField('update_mask'):
        raise exceptions.InputException(
            '`update_mask` must be set for %s delta.' % approval_name)
      elif not approval_delta.update_mask.IsValidForDescriptor(
          issue_objects_pb2.ApprovalValue.DESCRIPTOR):
        raise exceptions.InputException(
            'Invalid `update_mask` for %s delta.' % approval_name)
      filtered_value = issue_objects_pb2.ApprovalValue()
      approval_delta.update_mask.MergeMessage(
          approval_delta.approval_value,
          filtered_value,
          replace_message_field=True,
          replace_repeated_field=True)
      status = _APPROVAL_STATUS_INGEST[filtered_value.status]
      # Approvers
      # No autocreate.
      # A user may try to remove all existing approvers [a, b] and add another
      # approver [c]. If they mis-type `c` and we auto-create `c` instead of
      # raising error, this would cause the ApprovalValue to be editable by no
      # one but site admins.
      approver_ids_add = rnc.IngestUserNames(
          self.cnxn, filtered_value.approvers, self.services, autocreate=False)
      approver_ids_remove = rnc.IngestUserNames(
          self.cnxn,
          approval_delta.approvers_remove,
          self.services,
          autocreate=False)

      # Field Values.
      config = self.services.config.GetProjectConfig(self.cnxn, project_id)
      approval_fds_by_id = {
          fd.field_id: fd
          for fd in config.field_defs
          if fd.field_type is tracker_pb2.FieldTypes.APPROVAL_TYPE
      }
      if approval_id not in approval_fds_by_id:
        raise exceptions.InputException(
            'Approval not found in project for %s' % approval_name)

      sub_fvs_add, add_enums = self._IngestFieldValues(
          filtered_value.field_values, config, approval_id_filter=approval_id)
      sub_fvs_remove, remove_enums = self._IngestFieldValues(
          approval_delta.field_vals_remove,
          config,
          approval_id_filter=approval_id)
      labels_add = []
      labels_remove = []
      field_helpers.ShiftEnumFieldsIntoLabels(
          labels_add, labels_remove, add_enums, remove_enums, config)
      assert len(add_enums) == 0  # ShiftEnumFieldsIntoLabels clears all enums.
      assert len(remove_enums) == 0
      delta = tbo.MakeApprovalDelta(
          status,
          setter_id,
          approver_ids_add,
          approver_ids_remove,
          sub_fvs_add,
          sub_fvs_remove, [],
          labels_add,
          labels_remove,
          set_on=set_on)
      delta_specifications.append((issue_id, approval_id, delta))
    return delta_specifications

  def IngestIssue(self, issue, project_id):
    # type: (api_proto.issue_objects_pb2.Issue, int) -> proto.tracker_pb2.Issue
    """Ingest a protoc Issue into a protorpc Issue.

    Args:
      issue: the protoc issue to ingest.
      project_id: The project into which we're ingesting `issue`.

    Returns:
      protorpc version of issue, ignoring all OUTPUT_ONLY fields.

    Raises:
      InputException: if any fields in the 'issue' proto were invalid.
      NoSuchProjectException: if 'project_id' is not found.
    """
    # Get config first. We can't ingest the issue if the project isn't found.
    config = self.services.config.GetProjectConfig(self.cnxn, project_id)
    ingestedDict = {
      'project_id': project_id,
      'summary': issue.summary
    }
    with exceptions.ErrorAggregator(exceptions.InputException) as err_agg:
      self._ExtractOwner(issue, ingestedDict, err_agg)

      # Extract ccs.
      try:
        ingestedDict['cc_ids'] = rnc.IngestUserNames(
            self.cnxn, [cc.user for cc in issue.cc_users], self.services,
            autocreate=True)
      except exceptions.InputException as e:
        err_agg.AddErrorMessage('Error ingesting cc_users: {}', e)

      # Extract status.
      if issue.HasField('status') and issue.status.status:
        ingestedDict['status'] = issue.status.status
      else:
        err_agg.AddErrorMessage('Status is required when creating an issue')

      # Extract components.
      try:
        project_comp_ids = rnc.IngestComponentDefNames(
            self.cnxn, [cv.component for cv in issue.components], self.services)
        ingestedDict['component_ids'] = [
            comp_id for (_pid, comp_id) in project_comp_ids]
      except (exceptions.InputException, exceptions.NoSuchProjectException,
              exceptions.NoSuchComponentException) as e:
        err_agg.AddErrorMessage('Error ingesting components: {}', e)

      # Extract labels and field values.
      ingestedDict['labels'] = [lv.label for lv in issue.labels]
      try:
        ingestedDict['field_values'], enums = self._IngestFieldValues(
            issue.field_values, config)
        field_helpers.ShiftEnumFieldsIntoLabels(
            ingestedDict['labels'], [], enums, [], config)
        assert len(
            enums) == 0  # ShiftEnumFieldsIntoLabels must clear all enums.
      except exceptions.InputException as e:
        err_agg.AddErrorMessage(e.message)

      # Ingest merged, blocking/blocked_on.
      self._ExtractIssueRefs(issue, ingestedDict, err_agg)
    return tracker_pb2.Issue(**ingestedDict)

  def _IngestFieldValues(self, field_values, config, approval_id_filter=None):
    # type: (Sequence[api_proto.issue_objects.FieldValue],
    #     proto.tracker_pb2.ProjectIssueConfig, Optional[int]) ->
    #     Tuple[Sequence[proto.tracker_pb2.FieldValue],
    #         Mapping[int, Sequence[str]]]
    """Returns protorpc FieldValues for the given protoc FieldValues.

    Raises exceptions if any field could not be parsed for any reasons such as
        unsupported field type, non-existent field, field from different
        projects, or fields with mismatched parent approvals.

    Args:
      field_values: protoc FieldValues to ingest.
      config: ProjectIssueConfig for the FieldValues we're ingesting.
      approval_id_filter: an approval_id, including any FieldValues that does
          not have this approval as a parent will trigger InputException.

    Returns:
      A pair 1) Ingested FieldValues. 2) A mapping of field ids to values
      for ENUM_TYPE fields in 'field_values.'

    Raises:
      InputException: if any fields_values could not be parsed for any reasons
          such as unsupported field type, non-existent field, or field from
          different projects.
    """
    fds_by_id = {fd.field_id: fd for fd in config.field_defs}
    enums = {}
    ingestedFieldValues = []
    with exceptions.ErrorAggregator(exceptions.InputException) as err_agg:
      for fv in field_values:
        try:
          project_id, fd_id = rnc.IngestFieldDefName(
              self.cnxn, fv.field, self.services)
          fd = fds_by_id[fd_id]
          # Raise if field does not belong to approval_id_filter (if provided).
          if (approval_id_filter is not None and
              fd.approval_id != approval_id_filter):
            approval_name = rnc.ConvertApprovalDefNames(
                self.cnxn, [approval_id_filter], project_id,
                self.services)[approval_id_filter]
            err_agg.AddErrorMessage(
                'Field {} does not belong to approval {}', fv.field,
                approval_name)
            continue
          if fd.field_type == tracker_pb2.FieldTypes.ENUM_TYPE:
            enums.setdefault(fd_id, []).append(fv.value)
          else:
            ingestedFieldValues.append(self._IngestFieldValue(fv, fd))
        except (exceptions.InputException, exceptions.NoSuchProjectException,
                exceptions.NoSuchFieldDefException, ValueError) as e:
          err_agg.AddErrorMessage(
              'Could not ingest value ({}) for FieldDef ({}): {}', fv.value,
              fv.field, e)
        except exceptions.NoSuchUserException as e:
          err_agg.AddErrorMessage(
              'User ({}) not found when ingesting user field: {}', fv.value,
              fv.field)
        except KeyError as e:
          err_agg.AddErrorMessage('Field {} is not in this project', fv.field)
    return ingestedFieldValues, enums

  def _IngestFieldValue(self, field_value, field_def):
    # type: (api_proto.issue_objects.FieldValue, proto.tracker_pb2.FieldDef) ->
    #     proto.tracker_pb2.FieldValue
    """Ingest a protoc FieldValue into a protorpc FieldValue.

    Args:
      field_value: protoc FieldValue to ingest.
      field_def: protorpc FieldDef associated with 'field_value'.
          BOOL_TYPE and APPROVAL_TYPE are ignored.
          Enum values are not allowed. They must be ingested as labels.

    Returns:
      Ingested protorpc FieldValue.

    Raises:
      InputException if 'field_def' is USER_TYPE and 'field_value' does not
          have a valid formatted resource name.
      NoSuchUserException if specified user in field does not exist.
      ValueError if 'field_value' could not be parsed for 'field_def'.
    """
    assert field_def.field_type != tracker_pb2.FieldTypes.ENUM_TYPE
    if field_def.field_type == tracker_pb2.FieldTypes.USER_TYPE:
      return self._ParseOneUserFieldValue(field_value.value, field_def.field_id)
    fv = field_helpers.ParseOneFieldValue(
        self.cnxn, self.services.user, field_def, field_value.value)
    # ParseOneFieldValue currently ignores parsing errors, although it has TODOs
    # to raise them.
    if not fv:
      raise ValueError('Could not parse %s' % field_value.value)
    return fv

  def _ParseOneUserFieldValue(self, value, field_id):
    # type: (str, int) -> proto.tracker_pb2.FieldValue
    """Replacement for the obsolete user parsing in ParseOneFieldValue."""
    user_id = rnc.IngestUserName(self.cnxn, value, self.services)
    return tbo.MakeFieldValue(field_id, None, None, user_id, None, None, False)

  def _ExtractOwner(self, issue, ingestedDict, err_agg):
    # type: (api_proto.issue_objects_pb2.Issue, Dict[str, Any], ErrorAggregator)
    #     -> None
    """Fills 'owner' into `ingestedDict`, if it can be extracted."""
    if issue.HasField('owner'):
      try:
        # Unlike for cc's, we require owner be an existing user, thus call we
        # do not autocreate.
        ingestedDict['owner_id'] = rnc.IngestUserName(
            self.cnxn, issue.owner.user, self.services, autocreate=False)
      except exceptions.InputException as e:
        err_agg.AddErrorMessage(
            'Error ingesting owner ({}): {}', issue.owner.user, e)
      except exceptions.NoSuchUserException as e:
        err_agg.AddErrorMessage(
            'User ({}) not found when ingesting owner', e)
    else:
      ingestedDict['owner_id'] = framework_constants.NO_USER_SPECIFIED

  def _ExtractIssueRefs(self, issue, ingestedDict, err_agg):
    # type: (api_proto.issue_objects_pb2.Issue, Dict[str, Any], ErrorAggregator)
    #     -> None
    """Fills issue relationships into `ingestedDict` from `issue`."""
    if issue.HasField('merged_into_issue_ref'):
      try:
        merged_into_ref = self._IngestIssueRef(issue.merged_into_issue_ref)
        if isinstance(merged_into_ref, tracker_pb2.DanglingIssueRef):
          ingestedDict['merged_into_external'] = (
              merged_into_ref.ext_issue_identifier)
        else:
          ingestedDict['merged_into'] = merged_into_ref
      except exceptions.InputException as e:
        err_agg.AddErrorMessage(
            'Error ingesting ref {}: {}', issue.merged_into_issue_ref, e)
    try:
      iids, dangling_refs = self._IngestIssueRefs(issue.blocked_on_issue_refs)
      ingestedDict['blocked_on_iids'] = iids
      ingestedDict['dangling_blocked_on_refs'] = dangling_refs
    except exceptions.InputException as e:
      err_agg.AddErrorMessage(e.message)
    try:
      iids, dangling_refs = self._IngestIssueRefs(issue.blocking_issue_refs)
      ingestedDict['blocking_iids'] = iids
      ingestedDict['dangling_blocking_refs'] = dangling_refs
    except exceptions.InputException as e:
      err_agg.AddErrorMessage(e.message)

  def _IngestIssueRefs(self, issue_refs):
    # type: (api_proto.issue_objects.IssueRf) ->
    #     Tuple[Sequence[int], Sequence[tracker_pb2.DanglingIssueRef]]
    """Given protoc IssueRefs, returns issue_ids and DanglingIssueRefs."""
    issue_ids = []
    external_refs = []
    with exceptions.ErrorAggregator(exceptions.InputException) as err_agg:
      for ref in issue_refs:
        try:
          ingested_ref = self._IngestIssueRef(ref)
          if isinstance(ingested_ref, tracker_pb2.DanglingIssueRef):
            external_refs.append(ingested_ref)
          else:
            issue_ids.append(ingested_ref)
        except (exceptions.InputException, exceptions.NoSuchIssueException,
                exceptions.NoSuchProjectException) as e:
          err_agg.AddErrorMessage('Error ingesting ref {}: {}', ref, e)

    return issue_ids, external_refs

  def _IngestIssueRef(self, issue_ref):
    # type: (api_proto.issue_objects.IssueRef) ->
    #     Union[int, tracker_pb2.DanglingIssueRef]
    """Given a protoc IssueRef, returns an issue id or DanglingIssueRef."""
    if issue_ref.issue and issue_ref.ext_identifier:
      raise exceptions.InputException(
        'IssueRefs MUST NOT have both `issue` and `ext_identifier`')
    if issue_ref.issue:
      return rnc.IngestIssueName(self.cnxn, issue_ref.issue, self.services)
    if issue_ref.ext_identifier:
      # TODO(crbug.com/monorail/7208): Handle ingestion/conversion of CodeSite
      # refs. We may be able to avoid ever needing to ingest them.
      return tracker_pb2.DanglingIssueRef(
          ext_issue_identifier=issue_ref.ext_identifier
        )
    raise exceptions.InputException(
        'IssueRefs MUST have one of `issue` and `ext_identifier`')

  def IngestIssuesListColumns(self, issues_list_columns):
    # type: (Sequence[proto.issue_objects_pb2.IssuesListColumn] -> str
    """Ingest a list of protoc IssueListColumns and returns a string."""
    return ' '.join([col.column for col in issues_list_columns])

  def _ComputeIssuesListColumns(self, columns):
    # type: (string) -> Sequence[api_proto.issue_objects_pb2.IssuesListColumn]
    """Convert string representation of columns to protoc IssuesListColumns"""
    return [
        issue_objects_pb2.IssuesListColumn(column=col)
        for col in columns.split()
    ]

  def IngestNotifyType(self, notify):
    # type: (issue_pb.NotifyType) -> bool
    """Ingest a NotifyType to boolean."""
    if (notify == issues_pb2.NotifyType.Value('NOTIFY_TYPE_UNSPECIFIED') or
        notify == issues_pb2.NotifyType.Value('EMAIL')):
      return True
    elif notify == issues_pb2.NotifyType.Value('NO_NOTIFICATION'):
      return False

  # Users

  def ConvertUser(self, user):
    # type: (protorpc.User) -> api_proto.user_objects_pb2.User
    """Convert a protorpc User into a protoc User.

    Args:
      user: protorpc User object.

    Returns:
      The protoc User object.
    """
    return self.ConvertUsers([user.user_id])[user.user_id]


  # TODO(crbug/monorail/7238): Make this take in a full User object and
  # return a Sequence, rather than a map, after hotlist users are converted.
  def ConvertUsers(self, user_ids):
    # type: (Sequence[int]) -> Map(int, api_proto.user_objects_pb2.User)
    """Convert list of protorpc Users into list of protoc Users.

    Args:
      user_ids: List of User IDs.

    Returns:
      Dict of User IDs to User protos for given user_ids that could be found.
    """
    user_ids_to_names = {}

    # Get display names
    users_by_id = self.services.user.GetUsersByIDs(self.cnxn, user_ids)
    (display_names_by_id,
     display_emails_by_id) = framework_bizobj.CreateUserDisplayNamesAndEmails(
         self.cnxn, self.services, self.user_auth, users_by_id.values())

    for user_id, user in users_by_id.items():
      name = rnc.ConvertUserNames([user_id]).get(user_id)

      display_name = display_names_by_id.get(user_id)
      display_email = display_emails_by_id.get(user_id)
      availability = framework_helpers.GetUserAvailability(user)
      availability_message, _availability_status = availability

      user_ids_to_names[user_id] = user_objects_pb2.User(
          name=name,
          display_name=display_name,
          email=display_email,
          availability_message=availability_message)

    return user_ids_to_names

  def ConvertProjectStars(self, user_id, projects):
    # type: (int, Collection[protorpc.Project]) ->
    #     Collection[api_proto.user_objects_pb2.ProjectStar]
    """Convert list of protorpc Projects into protoc ProjectStars.

    Args:
      user_id: The user the ProjectStar is associated with.
      projects: All starred projects.

    Returns:
      List of ProjectStar messages.
    """
    api_project_stars = []
    for proj in projects:
      name = rnc.ConvertProjectStarName(
          self.cnxn, user_id, proj.project_id, self.services)
      star = user_objects_pb2.ProjectStar(name=name)
      api_project_stars.append(star)
    return api_project_stars

  # Field Defs

  def ConvertFieldDefs(self, field_defs, project_id):
    # type: (Sequence[proto.tracker_pb2.FieldDef], int) ->
    #     Sequence[api_proto.project_objects_pb2.FieldDef]
    """Convert sequence of protorpc FieldDefs to protoc FieldDefs.

    Args:
      field_defs: List of protorpc FieldDefs
      project_id: ID of the Project that is ancestor to all given
        `field_defs`.

    Returns:
      Sequence of protoc FieldDef in the same order they are given in
      `field_defs`. In the event any field_def or the referenced approval_id
      in `field_defs` is not found, they will be omitted from the result.
    """
    field_ids = [fd.field_id for fd in field_defs]
    resource_names_dict = rnc.ConvertFieldDefNames(
        self.cnxn, field_ids, project_id, self.services)
    parent_approval_ids = [
        fd.approval_id for fd in field_defs if fd.approval_id is not None
    ]
    approval_names_dict = rnc.ConvertApprovalDefNames(
        self.cnxn, parent_approval_ids, project_id, self.services)

    api_fds = []
    for fd in field_defs:
      # Skip over approval fields, they have their separate ApprovalDef
      if fd.field_type == tracker_pb2.FieldTypes.APPROVAL_TYPE:
        continue
      if fd.field_id not in resource_names_dict:
        continue

      name = resource_names_dict.get(fd.field_id)
      display_name = fd.field_name
      docstring = fd.docstring
      field_type = self._ConvertFieldDefType(fd.field_type)
      applicable_issue_type = fd.applicable_type
      admins = rnc.ConvertUserNames(fd.admin_ids).values()
      editors = rnc.ConvertUserNames(fd.editor_ids).values()
      traits = self._ComputeFieldDefTraits(fd)
      approval_parent = approval_names_dict.get(fd.approval_id)

      enum_settings = None
      if field_type == project_objects_pb2.FieldDef.Type.Value('ENUM'):
        enum_settings = project_objects_pb2.FieldDef.EnumTypeSettings(
            choices=self._GetEnumFieldChoices(fd))

      int_settings = None
      if field_type == project_objects_pb2.FieldDef.Type.Value('INT'):
        int_settings = project_objects_pb2.FieldDef.IntTypeSettings(
            min_value=fd.min_value, max_value=fd.max_value)

      str_settings = None
      if field_type == project_objects_pb2.FieldDef.Type.Value('STR'):
        str_settings = project_objects_pb2.FieldDef.StrTypeSettings(
            regex=fd.regex)

      user_settings = None
      if field_type == project_objects_pb2.FieldDef.Type.Value('USER'):
        user_settings = project_objects_pb2.FieldDef.UserTypeSettings(
            role_requirements=self._ConvertRoleRequirements(fd.needs_member),
            notify_triggers=self._ConvertNotifyTriggers(fd.notify_on),
            grants_perm=fd.grants_perm,
            needs_perm=fd.needs_perm)

      date_settings = None
      if field_type == project_objects_pb2.FieldDef.Type.Value('DATE'):
        date_settings = project_objects_pb2.FieldDef.DateTypeSettings(
            date_action=self._ConvertDateAction(fd.date_action))

      api_fd = project_objects_pb2.FieldDef(
          name=name,
          display_name=display_name,
          docstring=docstring,
          type=field_type,
          applicable_issue_type=applicable_issue_type,
          admins=admins,
          traits=traits,
          approval_parent=approval_parent,
          enum_settings=enum_settings,
          int_settings=int_settings,
          str_settings=str_settings,
          user_settings=user_settings,
          date_settings=date_settings,
          editors=editors)
      api_fds.append(api_fd)
    return api_fds

  def _ConvertDateAction(self, date_action):
    # type: (proto.tracker_pb2.DateAction) ->
    #     api_proto.project_objects_pb2.FieldDef.DateTypeSettings.DateAction
    """Convert protorpc DateAction to protoc
       FieldDef.DateTypeSettings.DateAction"""
    if date_action == tracker_pb2.DateAction.NO_ACTION:
      return project_objects_pb2.FieldDef.DateTypeSettings.DateAction.Value(
          'NO_ACTION')
    elif date_action == tracker_pb2.DateAction.PING_OWNER_ONLY:
      return project_objects_pb2.FieldDef.DateTypeSettings.DateAction.Value(
          'NOTIFY_OWNER')
    elif date_action == tracker_pb2.DateAction.PING_PARTICIPANTS:
      return project_objects_pb2.FieldDef.DateTypeSettings.DateAction.Value(
          'NOTIFY_PARTICIPANTS')
    else:
      raise ValueError('Unsupported DateAction Value')

  def _ConvertRoleRequirements(self, needs_member):
    # type: (bool) ->
    #     api_proto.project_objects_pb2.FieldDef.
    #     UserTypeSettings.RoleRequirements
    """Convert protorpc RoleRequirements to protoc
       FieldDef.UserTypeSettings.RoleRequirements"""

    proto_user_settings = project_objects_pb2.FieldDef.UserTypeSettings
    if needs_member:
      return proto_user_settings.RoleRequirements.Value('PROJECT_MEMBER')
    else:
      return proto_user_settings.RoleRequirements.Value('NO_ROLE_REQUIREMENT')

  def _ConvertNotifyTriggers(self, notify_trigger):
    # type: (proto.tracker_pb2.NotifyTriggers) ->
    #     api_proto.project_objects_pb2.FieldDef.UserTypeSettings.NotifyTriggers
    """Convert protorpc NotifyTriggers to protoc
       FieldDef.UserTypeSettings.NotifyTriggers"""
    if notify_trigger == tracker_pb2.NotifyTriggers.NEVER:
      return project_objects_pb2.FieldDef.UserTypeSettings.NotifyTriggers.Value(
          'NEVER')
    elif notify_trigger == tracker_pb2.NotifyTriggers.ANY_COMMENT:
      return project_objects_pb2.FieldDef.UserTypeSettings.NotifyTriggers.Value(
          'ANY_COMMENT')
    else:
      raise ValueError('Unsupported NotifyTriggers Value')

  def _ConvertFieldDefType(self, field_type):
    # type: (proto.tracker_pb2.FieldTypes) ->
    #     api_proto.project_objects_pb2.FieldDef.Type
    """Convert protorpc FieldType to protoc FieldDef.Type

    Args:
      field_type: protorpc FieldType

    Returns:
      Corresponding protoc FieldDef.Type

    Raises:
      ValueError if input `field_type` has no suitable supported FieldDef.Type,
      or input `field_type` is not a recognized enum option.
    """
    if field_type == tracker_pb2.FieldTypes.ENUM_TYPE:
      return project_objects_pb2.FieldDef.Type.Value('ENUM')
    elif field_type == tracker_pb2.FieldTypes.INT_TYPE:
      return project_objects_pb2.FieldDef.Type.Value('INT')
    elif field_type == tracker_pb2.FieldTypes.STR_TYPE:
      return project_objects_pb2.FieldDef.Type.Value('STR')
    elif field_type == tracker_pb2.FieldTypes.USER_TYPE:
      return project_objects_pb2.FieldDef.Type.Value('USER')
    elif field_type == tracker_pb2.FieldTypes.DATE_TYPE:
      return project_objects_pb2.FieldDef.Type.Value('DATE')
    elif field_type == tracker_pb2.FieldTypes.URL_TYPE:
      return project_objects_pb2.FieldDef.Type.Value('URL')
    else:
      raise ValueError(
          'Unsupported tracker_pb2.FieldType enum. Boolean types '
          'are unsupported and approval types are found in ApprovalDefs')

  def _ComputeFieldDefTraits(self, field_def):
    # type: (proto.tracker_pb2.FieldDef) ->
    #     Sequence[api_proto.project_objects_pb2.FieldDef.Traits]
    """Compute sequence of FieldDef.Traits for a given protorpc FieldDef."""
    trait_protos = []
    if field_def.is_required:
      trait_protos.append(project_objects_pb2.FieldDef.Traits.Value('REQUIRED'))
    if field_def.is_niche:
      trait_protos.append(
          project_objects_pb2.FieldDef.Traits.Value('DEFAULT_HIDDEN'))
    if field_def.is_multivalued:
      trait_protos.append(
          project_objects_pb2.FieldDef.Traits.Value('MULTIVALUED'))
    if field_def.is_phase_field:
      trait_protos.append(project_objects_pb2.FieldDef.Traits.Value('PHASE'))
    if field_def.is_restricted_field:
      trait_protos.append(
          project_objects_pb2.FieldDef.Traits.Value('RESTRICTED'))
    return trait_protos

  def _GetEnumFieldChoices(self, field_def):
    # type: (proto.tracker_pb2.FieldDef) ->
    #     Sequence[Choice]
    """Get sequence of choices for an enum field

    Args:
      field_def: protorpc FieldDef

    Returns:
      Sequence of valid Choices for enum field `field_def`.

    Raises:
      ValueError if input `field_def` is not an enum type field.
    """
    if field_def.field_type != tracker_pb2.FieldTypes.ENUM_TYPE:
      raise ValueError('Cannot get value from label for non-enum-type field')

    config = self.services.config.GetProjectConfig(
        self.cnxn, field_def.project_id)
    value_docstr_tuples = tracker_helpers._GetEnumFieldValuesAndDocstrings(
        field_def, config)

    return [
        Choice(value=value, docstring=docstring)
        for value, docstring in value_docstr_tuples
    ]

  # Field Values

  def _GetNonApprovalFieldValues(self, field_values, project_id):
    # type: (Sequence[proto.tracker_pb2.FieldValue], int) ->
    #     Sequence[proto.tracker_pb2.FieldValue]
    """Filter out field values that belong to an approval field."""
    config = self.services.config.GetProjectConfig(self.cnxn, project_id)
    approval_fd_ids = set(
        [fd.field_id for fd in config.field_defs if fd.approval_id])

    return [fv for fv in field_values if fv.field_id not in approval_fd_ids]

  def ConvertFieldValues(self, field_values, project_id, phases):
    # type: (Sequence[proto.tracker_pb2.FieldValue], int,
    #     Sequence[proto.tracker_pb2.Phase]) ->
    #     Sequence[api_proto.issue_objects_pb2.FieldValue]
    """Convert sequence of field_values to protoc FieldValues.

    This method does not handle enum_type fields.

    Args:
      field_values: List of FieldValues
      project_id: ID of the Project that is ancestor to all given
        `field_values`.
      phases: List of Phases

    Returns:
      Sequence of protoc FieldValues in the same order they are given in
      `field_values`. In the event any field_values in `field_values` are not
      found, they will be omitted from the result.
    """
    phase_names_by_id = {phase.phase_id: phase.name for phase in phases}
    field_ids = [fv.field_id for fv in field_values]
    resource_names_dict = rnc.ConvertFieldDefNames(
        self.cnxn, field_ids, project_id, self.services)

    api_fvs = []
    for fv in field_values:
      if fv.field_id not in resource_names_dict:
        continue

      name = resource_names_dict.get(fv.field_id)
      value = self._ComputeFieldValueString(fv)
      derivation = self._ComputeFieldValueDerivation(fv)
      phase = phase_names_by_id.get(fv.phase_id)
      api_item = issue_objects_pb2.FieldValue(
          field=name, value=value, derivation=derivation, phase=phase)
      api_fvs.append(api_item)

    return api_fvs

  def _ComputeFieldValueString(self, field_value):
    # type: (proto.tracker_pb2.FieldValue) -> str
    """Convert a FieldValue's value to a string."""
    if field_value is None:
      raise exceptions.InputException('No FieldValue specified')
    elif field_value.int_value is not None:
      return str(field_value.int_value)
    elif field_value.str_value is not None:
      return field_value.str_value
    elif field_value.user_id is not None:
      return rnc.ConvertUserNames([field_value.user_id
                                  ]).get(field_value.user_id)
    elif field_value.date_value is not None:
      return str(field_value.date_value)
    elif field_value.url_value is not None:
      return field_value.url_value
    else:
      raise exceptions.InputException('FieldValue must have at least one value')

  def _ComputeFieldValueDerivation(self, field_value):
    # type: (proto.tracker_pb2.FieldValue) ->
    #     api_proto.issue_objects_pb2.Issue.Derivation
    """Convert a FieldValue's 'derived' to a protoc Issue.Derivation.

    Args:
      field_value: protorpc FieldValue

    Returns:
      Issue.Derivation of given `field_value`
    """
    if field_value.derived:
      return issue_objects_pb2.Derivation.Value('RULE')
    else:
      return issue_objects_pb2.Derivation.Value('EXPLICIT')

  # Approval Def

  def ConvertApprovalDefs(self, approval_defs, project_id):
    # type: (Sequence[proto.tracker_pb2.ApprovalDef], int) ->
    #     Sequence[api_proto.project_objects_pb2.ApprovalDef]
    """Convert sequence of protorpc ApprovalDefs to protoc ApprovalDefs.

    Args:
      approval_defs: List of protorpc ApprovalDefs
      project_id: ID of the Project the approval_defs belong to.

    Returns:
      Sequence of protoc ApprovalDefs in the same order they are given in
      in `approval_defs`. In the event any approval_def in `approval_defs`
      are not found, they will be omitted from the result.
    """
    approval_ids = set([ad.approval_id for ad in approval_defs])
    resource_names_dict = rnc.ConvertApprovalDefNames(
        self.cnxn, approval_ids, project_id, self.services)

    # Get matching field defs, needed to fill out protoc ApprovalDefs
    config = self.services.config.GetProjectConfig(self.cnxn, project_id)
    fd_by_id = {}
    for fd in config.field_defs:
      if (fd.field_type == tracker_pb2.FieldTypes.APPROVAL_TYPE and
          fd.field_id in approval_ids):
        fd_by_id[fd.field_id] = fd

    all_users = tbo.UsersInvolvedInApprovalDefs(
        approval_defs, fd_by_id.values())
    user_resource_names_dict = rnc.ConvertUserNames(all_users)

    api_ads = []
    for ad in approval_defs:
      if (ad.approval_id not in resource_names_dict or
          ad.approval_id not in fd_by_id):
        continue
      matching_fd = fd_by_id.get(ad.approval_id)
      name = resource_names_dict.get(ad.approval_id)
      display_name = matching_fd.field_name
      docstring = matching_fd.docstring
      survey = ad.survey
      approvers = [
          user_resource_names_dict.get(approver_id)
          for approver_id in ad.approver_ids
      ]
      admins = [
          user_resource_names_dict.get(admin_id)
          for admin_id in matching_fd.admin_ids
      ]

      api_ad = project_objects_pb2.ApprovalDef(
          name=name,
          display_name=display_name,
          docstring=docstring,
          survey=survey,
          approvers=approvers,
          admins=admins)
      api_ads.append(api_ad)
    return api_ads

  def ConvertApprovalValues(self, approval_values, field_values, phases,
                            issue_id=None, project_id=None):
    # type: (Sequence[proto.tracker_pb2.ApprovalValue],
    #     Sequence[proto.tracker_pb2.FieldValue],
    #     Sequence[proto.tracker_pb2.Phase], Optional[int], Optional[int]) ->
    #     Sequence[api_proto.issue_objects_pb2.ApprovalValue]
    """Convert sequence of approval_values to protoc ApprovalValues.

    `approval_values` may belong to a template or an issue. If they belong to a
    template, `project_id` should be given for the project the template is in.
    If these are issue `approval_values` `issue_id` should be given`.
    So, one of `issue_id` or `project_id` must be provided.
    If both are given, we ignore `project_id` and assume the `approval_values`
    belong to an issue.

    Args:
      approval_values: List of ApprovalValues.
      field_values: List of FieldValues that may belong to the approval_values.
      phases: List of Phases that may be associated with the approval_values.
      issue_id: ID of the Issue that the `approval_values` belong to.
      project_id: ID of the Project that the `approval_values`
        template belongs to.

    Returns:
      Sequence of protoc ApprovalValues in the same order they are given in
      in `approval_values`. In the event any approval definitions in
      `approval_values` are not found, they will be omitted from the result.

    Raises:
      InputException if neither `issue_id` nor `project_id` is given.
    """

    approval_ids = [av.approval_id for av in approval_values]
    resource_names_dict = {}
    if issue_id is not None:
      # Only issue approval_values have resource names.
      resource_names_dict = rnc.ConvertApprovalValueNames(
          self.cnxn, issue_id, self.services)
      project_id = self.services.issue.GetIssue(self.cnxn, issue_id).project_id
    elif project_id is None:
      raise exceptions.InputException(
          'One  `issue_id` or `project_id` must be given.')

    phase_names_by_id = {phase.phase_id: phase.name for phase in phases}
    ad_names_dict = rnc.ConvertApprovalDefNames(
        self.cnxn, approval_ids, project_id, self.services)

    # Organize the field values by the approval values they are
    # associated with.
    config = self.services.config.GetProjectConfig(self.cnxn, project_id)
    fds_by_id = {fd.field_id: fd for fd in config.field_defs}
    fvs_by_parent_approvals = collections.defaultdict(list)
    for fv in field_values:
      fd = fds_by_id.get(fv.field_id)
      if fd and fd.approval_id:
        fvs_by_parent_approvals[fd.approval_id].append(fv)

    api_avs = []
    for av in approval_values:
      # We only skip missing approval names if we are converting issue approval
      # values.
      if issue_id is not None and av.approval_id not in resource_names_dict:
        continue

      name = resource_names_dict.get(av.approval_id)
      approval_def = ad_names_dict.get(av.approval_id)
      approvers = rnc.ConvertUserNames(av.approver_ids).values()
      status = self._ComputeApprovalValueStatus(av.status)
      setter = rnc.ConvertUserName(av.setter_id)
      phase = phase_names_by_id.get(av.phase_id)

      field_values = self.ConvertFieldValues(
          fvs_by_parent_approvals[av.approval_id], project_id, phases)

      api_item = issue_objects_pb2.ApprovalValue(
          name=name,
          approval_def=approval_def,
          approvers=approvers,
          status=status,
          setter=setter,
          field_values=field_values,
          phase=phase)
      if av.set_on:
        api_item.set_time.FromSeconds(av.set_on)
      api_avs.append(api_item)

    return api_avs

  def _ComputeApprovalValueStatus(self, status):
    # type: (proto.tracker_pb2.ApprovalStatus) ->
    #     api_proto.issue_objects_pb2.Issue.ApprovalStatus
    """Convert a protorpc ApprovalStatus to a protoc Issue.ApprovalStatus."""
    try:
      return _APPROVAL_STATUS_CONVERT[status]
    except KeyError:
      raise ValueError('Unrecognized tracker_pb2.ApprovalStatus enum')

  # Projects

  def ConvertIssueTemplates(self, project_id, templates):
    # type: (int, Sequence[proto.tracker_pb2.TemplateDef]) ->
    #     Sequence[api_proto.project_objects_pb2.IssueTemplate]
    """Convert a Sequence of TemplateDefs to protoc IssueTemplates.

    Args:
      project_id: ID of the Project the templates belong to.
      templates: Sequence of TemplateDef protorpc objects.

    Returns:
      Sequence of protoc IssueTemplate in the same order they are given in
      `templates`. In the rare event that any templates are not found,
      they will be omitted from the result.
    """
    api_templates = []

    resource_names_dict = rnc.ConvertTemplateNames(
        self.cnxn, project_id, [template.template_id for template in templates],
        self.services)

    for template in templates:
      if template.template_id not in resource_names_dict:
        continue
      name = resource_names_dict.get(template.template_id)
      summary_must_be_edited = template.summary_must_be_edited
      template_privacy = self._ComputeTemplatePrivacy(template)
      default_owner = self._ComputeTemplateDefaultOwner(template)
      component_required = template.component_required
      admins = rnc.ConvertUserNames(template.admin_ids).values()
      issue = self._FillIssueFromTemplate(template, project_id)
      approval_values = self.ConvertApprovalValues(
          template.approval_values, template.field_values, template.phases,
          project_id=project_id)
      api_templates.append(
          project_objects_pb2.IssueTemplate(
              name=name,
              display_name=template.name,
              issue=issue,
              approval_values=approval_values,
              summary_must_be_edited=summary_must_be_edited,
              template_privacy=template_privacy,
              default_owner=default_owner,
              component_required=component_required,
              admins=admins))

    return api_templates

  def _FillIssueFromTemplate(self, template, project_id):
    # type: (proto.tracker_pb2.TemplateDef, int) ->
    #     api_proto.issue_objects_pb2.Issue
    """Convert a TemplateDef to its embedded protoc Issue.

    IssueTemplate does not set the following fields:
      name
      reporter
      cc_users
      blocked_on_issue_refs
      blocking_issue_refs
      create_time
      close_time
      modify_time
      component_modify_time
      status_modify_time
      owner_modify_time
      attachment_count
      star_count

    Args:
      template: TemplateDef protorpc objects.
      project_id: ID of the Project the template belongs to.

    Returns:
      protoc Issue filled with data from given `template`.
    """
    summary = template.summary
    state = issue_objects_pb2.IssueContentState.Value('ACTIVE')
    status = issue_objects_pb2.Issue.StatusValue(
        status=template.status,
        derivation=issue_objects_pb2.Derivation.Value('EXPLICIT'))
    owner = None
    if template.owner_id is not None:
      owner = issue_objects_pb2.Issue.UserValue(
          user=rnc.ConvertUserNames([template.owner_id]).get(template.owner_id))
    labels = self.ConvertLabels(template.labels, [], project_id)
    components_dict = rnc.ConvertComponentDefNames(
        self.cnxn, template.component_ids, project_id, self.services)
    components = []
    for component_resource_name in components_dict.values():
      components.append(
          issue_objects_pb2.Issue.ComponentValue(
              component=component_resource_name,
              derivation=issue_objects_pb2.Derivation.Value('EXPLICIT')))
    non_approval_fvs = self._GetNonApprovalFieldValues(
        template.field_values, project_id)
    field_values = self.ConvertFieldValues(
        non_approval_fvs, project_id, template.phases)
    field_values.extend(
        self.ConvertEnumFieldValues(template.labels, [], project_id))
    phases = self._ComputePhases(template.phases)

    filled_issue = issue_objects_pb2.Issue(
        summary=summary,
        state=state,
        status=status,
        owner=owner,
        labels=labels,
        components=components,
        field_values=field_values,
        phases=phases)
    return filled_issue

  def _ComputeTemplatePrivacy(self, template):
    # type: (proto.tracker_pb2.TemplateDef) ->
    #     api_proto.project_objects_pb2.IssueTemplate.TemplatePrivacy
    """Convert a protorpc TemplateDef to its protoc TemplatePrivacy."""
    if template.members_only:
      return project_objects_pb2.IssueTemplate.TemplatePrivacy.Value(
          'MEMBERS_ONLY')
    else:
      return project_objects_pb2.IssueTemplate.TemplatePrivacy.Value('PUBLIC')

  def _ComputeTemplateDefaultOwner(self, template):
    # type: (proto.tracker_pb2.TemplateDef) ->
    #     api_proto.project_objects_pb2.IssueTemplate.DefaultOwner
    """Convert a protorpc TemplateDef to its protoc DefaultOwner."""
    if template.owner_defaults_to_member:
      return project_objects_pb2.IssueTemplate.DefaultOwner.Value(
          'PROJECT_MEMBER_REPORTER')
    else:
      return project_objects_pb2.IssueTemplate.DefaultOwner.Value(
          'DEFAULT_OWNER_UNSPECIFIED')

  def _ComputePhases(self, phases):
    # type: (proto.tracker_pb2.TemplateDef) -> Sequence[str]
    """Convert a protorpc TemplateDef to its sorted string phases."""
    sorted_phases = sorted(phases, key=lambda phase: phase.rank)
    return [phase.name for phase in sorted_phases]

  def ConvertLabels(self, labels, derived_labels, project_id):
    # type: (Sequence[str], Sequence[str], int) ->
    #     Sequence[api_proto.issue_objects_pb2.Issue.LabelValue]
    """Convert string labels to LabelValues for non-enum-field labels

    Args:
      labels: Sequence of string labels
      project_id: ID of the Project these labels belong to.

    Return:
      Sequence of protoc IssueValues for given `labels` that
      do not represent enum field values.
    """
    config = self.services.config.GetProjectConfig(self.cnxn, project_id)
    non_fd_labels, non_fd_der_labels = tbo.ExplicitAndDerivedNonMaskedLabels(
        labels, derived_labels, config)
    api_labels = []
    for label in non_fd_labels:
      api_labels.append(
          issue_objects_pb2.Issue.LabelValue(
              label=label,
              derivation=issue_objects_pb2.Derivation.Value('EXPLICIT')))
    for label in non_fd_der_labels:
      api_labels.append(
          issue_objects_pb2.Issue.LabelValue(
              label=label,
              derivation=issue_objects_pb2.Derivation.Value('RULE')))
    return api_labels

  def ConvertEnumFieldValues(self, labels, derived_labels, project_id):
    # type: (Sequence[str], Sequence[str], int) ->
    #     Sequence[api_proto.issue_objects_pb2.FieldValue]
    """Convert string labels to FieldValues for enum-field labels

    Args:
      labels: Sequence of string labels
      project_id: ID of the Project these labels belong to.

    Return:
      Sequence of protoc FieldValues only for given `labels` that
      represent enum field values.
    """
    config = self.services.config.GetProjectConfig(self.cnxn, project_id)
    enum_ids_by_name = {
        fd.field_name.lower(): fd.field_id
        for fd in config.field_defs
        if fd.field_type is tracker_pb2.FieldTypes.ENUM_TYPE
        and not fd.is_deleted
    }
    resource_names_dict = rnc.ConvertFieldDefNames(
        self.cnxn, enum_ids_by_name.values(), project_id, self.services)

    api_fvs = []

    labels_by_prefix = tbo.LabelsByPrefix(labels, enum_ids_by_name.keys())
    for lower_field_name, values in labels_by_prefix.items():
      field_id = enum_ids_by_name.get(lower_field_name)
      resource_name = resource_names_dict.get(field_id)
      if not resource_name:
        continue
      api_fvs.extend(
          [
              issue_objects_pb2.FieldValue(
                  field=resource_name,
                  value=value,
                  derivation=issue_objects_pb2.Derivation.Value(
                      'EXPLICIT')) for value in values
          ])

    der_labels_by_prefix = tbo.LabelsByPrefix(
        derived_labels, enum_ids_by_name.keys())
    for lower_field_name, values in der_labels_by_prefix.items():
      field_id = enum_ids_by_name.get(lower_field_name)
      resource_name = resource_names_dict.get(field_id)
      if not resource_name:
        continue
      api_fvs.extend(
          [
              issue_objects_pb2.FieldValue(
                  field=resource_name,
                  value=value,
                  derivation=issue_objects_pb2.Derivation.Value('RULE'))
              for value in values
          ])

    return api_fvs

  def ConvertProject(self, project):
    # type: (proto.project_pb2.Project) ->
    #     api_proto.project_objects_pb2.Project
    """Convert a protorpc Project to its protoc Project."""

    return project_objects_pb2.Project(
        name=rnc.ConvertProjectName(
            self.cnxn, project.project_id, self.services),
        display_name=project.project_name,
        summary=project.summary,
        thumbnail_url=project_helpers.GetThumbnailUrl(project.logo_gcs_id))

  def ConvertProjects(self, projects):
    # type: (Sequence[proto.project_pb2.Project]) ->
    #     Sequence[api_proto.project_objects_pb2.Project]
    """Convert a Sequence of protorpc Projects to protoc Projects."""
    return [self.ConvertProject(proj) for proj in projects]

  def ConvertProjectConfig(self, project_config):
    # type: (proto.tracker_pb2.ProjectIssueConfig) ->
    #     api_proto.project_objects_pb2.ProjectConfig
    """Convert protorpc ProjectIssueConfig to protoc ProjectConfig."""
    project = self.services.project.GetProject(
        self.cnxn, project_config.project_id)
    project_grid_config = project_objects_pb2.ProjectConfig.GridViewConfig(
        default_x_attr=project_config.default_x_attr,
        default_y_attr=project_config.default_y_attr)
    template_names = rnc.ConvertTemplateNames(
        self.cnxn, project_config.project_id, [
            project_config.default_template_for_developers,
            project_config.default_template_for_users
        ], self.services)
    return project_objects_pb2.ProjectConfig(
        name=rnc.ConvertProjectConfigName(
            self.cnxn, project_config.project_id, self.services),
        exclusive_label_prefixes=project_config.exclusive_label_prefixes,
        member_default_query=project_config.member_default_query,
        default_sort=project_config.default_sort_spec,
        default_columns=self._ComputeIssuesListColumns(
            project_config.default_col_spec),
        project_grid_config=project_grid_config,
        member_default_template=template_names.get(
            project_config.default_template_for_developers),
        non_members_default_template=template_names.get(
            project_config.default_template_for_users),
        revision_url_format=project.revision_url_format,
        custom_issue_entry_url=project_config.custom_issue_entry_url)

  def CreateProjectMember(self, cnxn, project_id, user_id, role):
    # type: (MonorailContext, int, int, str) ->
    #     api_proto.project_objects_pb2.ProjectMember
    """Creates a ProjectMember object from specified parameters.

    Args:
      cnxn: MonorailConnection object.
      project_id: ID of the Project the User is a member of.
      user_id: ID of the user who is a member.
      role: str specifying the user's role based on a ProjectRole value.

    Return:
      A protoc ProjectMember object.
    """
    name = rnc.ConvertProjectMemberName(
        cnxn, project_id, user_id, self.services)
    return project_objects_pb2.ProjectMember(
        name=name,
        role=project_objects_pb2.ProjectMember.ProjectRole.Value(role))

  def ConvertLabelDefs(self, label_defs, project_id):
    # type: (Sequence[proto.tracker_pb2.LabelDef], int) ->
    #     Sequence[api_proto.project_objects_pb2.LabelDef]
    """Convert protorpc LabelDefs to protoc LabelDefs"""
    resource_names_dict = rnc.ConvertLabelDefNames(
        self.cnxn, [ld.label for ld in label_defs], project_id, self.services)

    api_lds = []
    for ld in label_defs:
      state = project_objects_pb2.LabelDef.LabelDefState.Value('ACTIVE')
      if ld.deprecated:
        state = project_objects_pb2.LabelDef.LabelDefState.Value('DEPRECATED')
      api_lds.append(
          project_objects_pb2.LabelDef(
              name=resource_names_dict.get(ld.label),
              value=ld.label,
              docstring=ld.label_docstring,
              state=state))
    return api_lds

  def ConvertStatusDefs(self, status_defs, project_id):
    # type: (Sequence[proto.tracker_pb2.StatusDef], int) ->
    #     Sequence[api_proto.project_objects_pb2.StatusDef]
    """Convert protorpc StatusDefs to protoc StatusDefs

    Args:
      status_defs: Sequence of StatusDefs.
      project_id: ID of the Project these belong to.

    Returns:
      Sequence of protoc StatusDefs in the same order they are given in
      `status_defs`.
    """
    resource_names_dict = rnc.ConvertStatusDefNames(
        self.cnxn, [sd.status for sd in status_defs], project_id, self.services)
    config = self.services.config.GetProjectConfig(self.cnxn, project_id)
    mergeable_statuses = set(config.statuses_offer_merge)

    # Rank is only surfaced as positional value in well_known_statuses
    rank_by_status = {}
    for rank, sd in enumerate(config.well_known_statuses):
      rank_by_status[sd.status] = rank

    api_sds = []
    for sd in status_defs:
      state = project_objects_pb2.StatusDef.StatusDefState.Value('ACTIVE')
      if sd.deprecated:
        state = project_objects_pb2.StatusDef.StatusDefState.Value('DEPRECATED')

      if sd.means_open:
        status_type = project_objects_pb2.StatusDef.StatusDefType.Value('OPEN')
      else:
        if sd.status in mergeable_statuses:
          status_type = project_objects_pb2.StatusDef.StatusDefType.Value(
              'MERGED')
        else:
          status_type = project_objects_pb2.StatusDef.StatusDefType.Value(
              'CLOSED')

      api_sd = project_objects_pb2.StatusDef(
          name=resource_names_dict.get(sd.status),
          value=sd.status,
          type=status_type,
          rank=rank_by_status[sd.status],
          docstring=sd.status_docstring,
          state=state,
      )
      api_sds.append(api_sd)
    return api_sds

  def ConvertComponentDef(self, component_def):
    # type: (proto.tracker_pb2.ComponentDef) ->
    #     api_proto.project_objects.ComponentDef
    """Convert a protorpc ComponentDef to a protoc ComponentDef."""
    return self.ConvertComponentDefs([component_def],
                                     component_def.project_id)[0]

  def ConvertComponentDefs(self, component_defs, project_id):
    # type: (Sequence[proto.tracker_pb2.ComponentDef], int) ->
    #     Sequence[api_proto.project_objects.ComponentDef]
    """Convert sequence of protorpc ComponentDefs to protoc ComponentDefs

    Args:
      component_defs: Sequence of protoc ComponentDefs.
      project_id: ID of the Project these belong to.

    Returns:
      Sequence of protoc ComponentDefs in the same order they are given in
      `component_defs`.
    """
    resource_names_dict = rnc.ConvertComponentDefNames(
        self.cnxn, [cd.component_id for cd in component_defs], project_id,
        self.services)
    involved_user_ids = tbo.UsersInvolvedInComponents(component_defs)
    user_resource_names_dict = rnc.ConvertUserNames(involved_user_ids)

    all_label_ids = set()
    for cd in component_defs:
      all_label_ids.update(cd.label_ids)

    # If this becomes a performance issue, we should add bulk look up.
    labels_by_id = {
        label_id: self.services.config.LookupLabel(
            self.cnxn, project_id, label_id) for label_id in all_label_ids
    }

    api_cds = []
    for cd in component_defs:
      state = project_objects_pb2.ComponentDef.ComponentDefState.Value('ACTIVE')
      if cd.deprecated:
        state = project_objects_pb2.ComponentDef.ComponentDefState.Value(
            'DEPRECATED')

      api_cd = project_objects_pb2.ComponentDef(
          name=resource_names_dict.get(cd.component_id),
          value=cd.path,
          docstring=cd.docstring,
          state=state,
          admins=[
              user_resource_names_dict.get(admin_id)
              for admin_id in cd.admin_ids
          ],
          ccs=[user_resource_names_dict.get(cc_id) for cc_id in cd.cc_ids],
          creator=user_resource_names_dict.get(cd.creator_id),
          modifier=user_resource_names_dict.get(cd.modifier_id),
          create_time=timestamp_pb2.Timestamp(seconds=cd.created),
          modify_time=timestamp_pb2.Timestamp(seconds=cd.modified),
          labels=[labels_by_id[label_id] for label_id in cd.label_ids],
      )
      api_cds.append(api_cd)
    return api_cds

  def ConvertProjectSavedQueries(self, saved_queries, project_id):
    # type: (Sequence[proto.tracker_pb2.SavedQuery], int) ->
    #     Sequence(api_proto.project_objects.ProjectSavedQuery)
    """Convert sequence of protorpc SavedQueries to protoc ProjectSavedQueries

    Args:
      saved_queries: Sequence of SavedQueries.
      project_id: ID of the Project these belong to.

    Returns:
      Sequence of protoc ProjectSavedQueries in the same order they are given in
      `saved_queries`. In the event any items in `saved_queries` are not found
      or don't belong to the project, they will be omitted from the result.
    """
    resource_names_dict = rnc.ConvertProjectSavedQueryNames(
        self.cnxn, [sq.query_id for sq in saved_queries], project_id,
        self.services)
    api_psqs = []
    for sq in saved_queries:
      if sq.query_id not in resource_names_dict:
        continue

      # TODO(crbug/monorail/7756): Remove base_query_id, avoid confusions.
      # Until then we have to expand the query by including base_query_id.
      # base_query_id can only be in the set of DEFAULT_CANNED_QUERIES.
      if sq.base_query_id:
        query = '{} {}'.format(tbo.GetBuiltInQuery(sq.base_query_id), sq.query)
      else:
        query = sq.query

      api_psqs.append(
          project_objects_pb2.ProjectSavedQuery(
              name=resource_names_dict.get(sq.query_id),
              display_name=sq.name,
              query=query))
    return api_psqs
