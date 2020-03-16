# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import logging

from google.protobuf import timestamp_pb2

from api import resource_name_converters as rnc
from api.v1.api_proto import feature_objects_pb2
from api.v1.api_proto import issue_objects_pb2
from api.v1.api_proto import user_objects_pb2

from framework import exceptions
from framework import framework_bizobj
from framework import framework_helpers
from proto import tracker_pb2


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
    # TODO(crbug/monorail/7238): Get the list of projects that have issues
    # in hotlist.
    members_by_id = self.ConvertUsers(
        hotlist.owner_ids + hotlist.editor_ids, None)
    default_columns = [
        issue_objects_pb2.IssuesListColumn(column=col)
        for col in hotlist.default_col_spec.split()
    ]
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
    # TODO(crbug/monorail/7238): Get the list of projects that have issues
    # in hotlist.
    adders_by_id = self.ConvertUsers([item.adder_id for item in items], None)

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

  def _ConvertStatusValue(self, issue):
    # proto.tracker_pb2.Issue -> api_proto.issue_objects_pb2.Issue.StatusValue
    """Convert the status string on issue into a StatusValue."""
    derivation = issue_objects_pb2.Issue.Derivation.Value(
        'DERIVATION_UNSPECIFIED')
    if not issue.status:
      derivation = issue_objects_pb2.Issue.Derivation.Value('RULE')
    else:
      derivation = issue_objects_pb2.Issue.Derivation.Value('EXPLICIT')
    return issue_objects_pb2.Issue.StatusValue(
        status=issue.status or issue.derived_status, derivation=derivation)

  def _ConvertLabelValues(self, issue):
    # proto.tracker_pb2.Issue ->
    #     Sequence[api_proto.issue_objects_pb2.Issue.LabelValue]
    """Convert the label strings on issue into LabelValues."""
    labels = []
    for label in issue.labels:
      labels.append(
          issue_objects_pb2.Issue.LabelValue(
              derivation=issue_objects_pb2.Issue.Derivation.Value('EXPLICIT'),
              label=label))
    for derived_label in issue.derived_labels:
      labels.append(
          issue_objects_pb2.Issue.LabelValue(
              derivation=issue_objects_pb2.Issue.Derivation.Value('RULE'),
              label=derived_label))
    return labels

  def ConvertIssues(self, issues):
    # type: (Sequence[proto.tracker_pb2.Issue]) ->
    #     Sequence[api_proto.issue_objects_pb2.Issue]
    """Convert protorpc Issues into protoc Issues."""
    issue_ids = [issue.issue_id for issue in issues]
    issue_names_dict = rnc.ConvertIssueNames(
        self.cnxn, issue_ids, self.services)
    # TODO(jessan): Assert that all issues are for the same project (config).
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
      labels = self._ConvertLabelValues(issue)
      converted_issues.append(
          issue_objects_pb2.Issue(
              name=issue_names_dict[issue.issue_id],
              summary=issue.summary,
              state=content_state,
              status=status,
              description='TODO(jessan): Pull description from comments',
              labels=labels,
              star_count=issue.star_count,
          ))
    return converted_issues

  def IngestIssuesListColumns(self, issues_list_columns):
    # type: (Sequence[proto.issue_objects_pb2.IssuesListColumn] -> str
    """Ingest a list of protoc IssueListColumns and returns a string."""
    return ' '.join([col.column for col in issues_list_columns])

  # Users

  # Because Monorail obscures emails of Users on the site, wherever
  # in the API we would normally use User resource names, we use
  # full User objects instead. For this reason, ConvertUsers is called
  # where we would normally call some ConvertUserResourceNames function.
  # So ConvertUsers follows the patterns in resource_name_converters.py
  # by taking in User IDs and and returning a dict rather than a list.
  # TODO(crbug/monorail/7238): take a list of projects when
  # CreateUserDisplayNames() can take in a list of projects.
  def ConvertUsers(self, user_ids, project):
    # type: (List(int), protorpc.Project) ->
    #   Map(int, api_proto.user_objects_pb2.User)
    """Convert list of protorpc Users into list of protoc Users.

    Args:
    user_ids: List of User IDs.
    project: currently viewed project.

    Returns:
      Dict of User IDs to User resource names for all given users.
    """
    user_ids_to_names = {}

    # Get display names
    users_by_id = self.services.user.GetUsersByIDs(self.cnxn, user_ids)
    display_names_by_id = framework_bizobj.CreateUserDisplayNames(
        self.user_auth, users_by_id.values(), project)

    for user_id, user in users_by_id.items():
      name = rnc.ConvertUserNames([user_id]).get(user_id)

      display_name = display_names_by_id.get(user_id)
      availability = framework_helpers.GetUserAvailability(user)
      availability_message, _availability_status = availability

      user_ids_to_names[user_id] = user_objects_pb2.User(
          name=name,
          display_name=display_name,
          availability_message=availability_message)

    return user_ids_to_names

  # Field Values

  def ConvertFieldValues(self, field_values, project_id, phases):
    # type: (Sequence[proto.tracker_pb2.FieldValue], int,
    #     Sequence[proto.tracker_pb2.Phase]) ->
    #     Sequence[api_proto.issue_objects_pb2.Issue.FieldValue]
    """Convert sequence of field_values to protoc FieldValues.

    This method does not handle enum_type fields

    Args:
      field_values: List of FieldValues
      project_id: ID of the Project that is ancestor to all given
        `field_values`.
      phases: List of Phases

    Returns:
      Sequence of protoc Issue.FieldValue in the same order they are given in
      `field_values`. In the event any field_values in `field_values` are not
      found, they will be omitted from the result.
    """
    phase_names_by_id = {phase.phase_id: phase.name for phase in phases}
    field_ids = [fv.field_id for fv in field_values]
    resource_names_dict = rnc.ConvertFieldDefNames(
        self.cnxn, field_ids, project_id, self.services)

    api_fvs = []
    for fv in field_values:
      # If the FieldDef with field_id was not found in ConvertFieldDefNames()
      # we skip
      if fv.field_id not in resource_names_dict:
        logging.info(
            'Ignoring field value referencing a non-existent field: %r', fv)
        continue

      name = resource_names_dict.get(fv.field_id)
      value = self._ComputeFieldValueString(fv)
      derivation = self._ComputeFieldValueDerivation(fv)
      phase = phase_names_by_id.get(fv.phase_id)
      api_item = issue_objects_pb2.Issue.FieldValue(
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
      return issue_objects_pb2.Issue.Derivation.Value('RULE')
    else:
      return issue_objects_pb2.Issue.Derivation.Value('EXPLICIT')

  def ConvertApprovalValues(self, approval_values, project_id, phases):
    # type: (Sequence[proto.tracker_pb2.ApprovalValue], int,
    #     Sequence[proto.tracker_pb2.Phase]) ->
    #     Sequence[api_proto.issue_objects_pb2.Issue.ApprovalValue]
    """Convert sequence of approval_values to protoc ApprovalValues.

    Args:
      approval_values: List of ApprovalValues
      project_id: ID of the Project that all given `approval_values` belong to.
      phases: List of Phases

    Returns:
      Sequence of protoc Issue.ApprovalValue in the same order they are given in
      in `approval_values`. In the event any approval_value in `approval_values`
      are not found, they will be omitted from the result.
    """
    phase_names_by_id = {phase.phase_id: phase.name for phase in phases}
    approval_ids = [av.approval_id for av in approval_values]
    resource_names_dict = rnc.ConvertApprovalDefNames(
        self.cnxn, approval_ids, project_id, self.services)

    api_avs = []
    for av in approval_values:
      # If the FieldDef with approval_id was not found in
      # ConvertApprovalDefNames(), we skip
      if av.approval_id not in resource_names_dict:
        logging.info(
            'Ignoring approval value referencing a non-existent field: %r', av)
        continue
      name = resource_names_dict.get(av.approval_id)
      approvers = rnc.ConvertUserNames(av.approver_ids).values()
      status = self._ComputeApprovalValueStatus(av.status)
      set_time = timestamp_pb2.Timestamp()
      set_time.FromSeconds(av.set_on)
      setter = rnc.ConvertUserNames([av.setter_id]).get(av.setter_id)
      phase = phase_names_by_id.get(av.phase_id)
      api_item = issue_objects_pb2.Issue.ApprovalValue(
          name=name,
          approvers=approvers,
          status=status,
          set_time=set_time,
          setter=setter,
          phase=phase)
      api_avs.append(api_item)

    return api_avs

  def _ComputeApprovalValueStatus(self, status):
    # type: (proto.tracker_pb2.ApprovalStatus) ->
    #     api_proto.issue_objects_pb2.Issue.ApprovalStatus
    """Convert a protorpc ApprovalStatus to a protoc Issue.ApprovalStatus."""
    if status == tracker_pb2.ApprovalStatus.NOT_SET:
      return issue_objects_pb2.Issue.ApprovalStatus.Value(
          'APPROVAL_STATUS_UNSPECIFIED')
    elif status == tracker_pb2.ApprovalStatus.NEEDS_REVIEW:
      return issue_objects_pb2.Issue.ApprovalStatus.Value('NEEDS_REVIEW')
    elif status == tracker_pb2.ApprovalStatus.NA:
      return issue_objects_pb2.Issue.ApprovalStatus.Value('NA')
    elif status == tracker_pb2.ApprovalStatus.REVIEW_REQUESTED:
      return issue_objects_pb2.Issue.ApprovalStatus.Value('REVIEW_REQUESTED')
    elif status == tracker_pb2.ApprovalStatus.REVIEW_STARTED:
      return issue_objects_pb2.Issue.ApprovalStatus.Value('REVIEW_STARTED')
    elif status == tracker_pb2.ApprovalStatus.NEED_INFO:
      return issue_objects_pb2.Issue.ApprovalStatus.Value('NEED_INFO')
    elif status == tracker_pb2.ApprovalStatus.APPROVED:
      return issue_objects_pb2.Issue.ApprovalStatus.Value('APPROVED')
    elif status == tracker_pb2.ApprovalStatus.NOT_APPROVED:
      return issue_objects_pb2.Issue.ApprovalStatus.Value('NOT_APPROVED')
    else:
      raise ValueError('Unrecognized tracker_pb2.ApprovalStatus enum')
