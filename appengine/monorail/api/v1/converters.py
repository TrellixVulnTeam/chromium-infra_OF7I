# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import itertools
import logging

from google.protobuf import timestamp_pb2

from api import resource_name_converters as rnc
from api.v1.api_proto import feature_objects_pb2
from api.v1.api_proto import issue_objects_pb2
from api.v1.api_proto import project_objects_pb2
from api.v1.api_proto import user_objects_pb2

from framework import exceptions
from framework import framework_bizobj
from framework import framework_helpers
from proto import tracker_pb2
from tracker import tracker_bizobj as tbo


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
                derivation=issue_objects_pb2.Issue.Derivation.Value(
                    'EXPLICIT')))
    for derived_component_id in issue.derived_component_ids:
      if derived_component_id in ids_to_names:
        component_values.append(
            issue_objects_pb2.Issue.ComponentValue(
                component=ids_to_names[derived_component_id],
                derivation=issue_objects_pb2.Issue.Derivation.Value('RULE')))

    return component_values

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
      components = self._ConvertComponentValues(issue)
      # TODO(jessan): Handle enum fieldvalues, then include below.
      # field_values = self.ConvertFieldValues(
      #     issue.field_values, issue.project_id, issue.phases)
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

      result = issue_objects_pb2.Issue(
          name=issue_names_dict[issue.issue_id],
          summary=issue.summary,
          state=content_state,
          status=status,
          description='TODO(jessan): Pull description from comments',
          # TODO(jessan): Add user fields.
          reporter=None,
          owner=None,
          cc_users=[],
          labels=labels,
          components=components,
          field_values=[],
          merged_into_issue_ref=merged_into_issue_ref,
          blocked_on_issue_refs=blocked_on_issue_refs,
          blocking_issue_refs=blocking_issue_refs,
          # TODO(jessan): add timestamps.
          star_count=issue.star_count)
      # TODO(crbug.com/monorail/5857): Set attachment_count unconditionally
      # after the underlying source of negative attachment counts has been
      # resolved and database has been repaired.
      if issue.attachment_count >= 0:
        result.attachment_count = issue.attachment_count
      converted_issues.append(result)
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
        logging.info(
            'Ignoring template referencing a non-existent template id: %s',
            template.template_id)
        continue
      name = resource_names_dict.get(template.template_id)
      summary_must_be_edited = template.summary_must_be_edited
      template_privacy = self._ComputeTemplatePrivacy(template)
      default_owner = self._ComputeTemplateDefaultOwner(template)
      component_required = template.component_required
      admins = rnc.ConvertUserNames(template.admin_ids).values()
      issue = self._FillIssueFromTemplate(template, project_id)
      api_templates.append(
          project_objects_pb2.IssueTemplate(
              name=name,
              issue=issue,
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
        derivation=issue_objects_pb2.Issue.Derivation.Value('EXPLICIT'))
    description = template.content
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
              derivation=issue_objects_pb2.Issue.Derivation.Value('EXPLICIT')))
    non_enum_field_values = self.ConvertFieldValues(
        template.field_values, project_id, template.phases)
    enum_field_values = self.ConvertEnumFieldValues(
        template.labels, [], project_id)
    field_values = non_enum_field_values + enum_field_values
    approval_values = self.ConvertApprovalValues(
        template.approval_values, project_id, template.phases)
    phases = self._ComputeTemplatePhases(template)

    filled_issue = issue_objects_pb2.Issue(
        summary=summary,
        state=state,
        status=status,
        description=description,
        owner=owner,
        labels=labels,
        components=components,
        field_values=field_values,
        approval_values=approval_values,
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

  def _ComputeTemplatePhases(self, template):
    # type: (proto.tracker_pb2.TemplateDef) -> Sequence[str]
    """Convert a protorpc TemplateDef to its sorted string phases."""
    sorted_phases = sorted(template.phases, key=lambda phase: phase.rank)
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
              derivation=issue_objects_pb2.Issue.Derivation.Value('EXPLICIT')))
    for label in non_fd_der_labels:
      api_labels.append(
          issue_objects_pb2.Issue.LabelValue(
              label=label,
              derivation=issue_objects_pb2.Issue.Derivation.Value('RULE')))
    return api_labels

  def ConvertEnumFieldValues(self, labels, derived_labels, project_id):
    # type: (Sequence[str], Sequence[str], int) ->
    #     Sequence[api_proto.issue_objects_pb2.Issue.FieldValue]
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
              issue_objects_pb2.Issue.FieldValue(
                  field=resource_name,
                  value=value,
                  derivation=issue_objects_pb2.Issue.Derivation.Value(
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
              issue_objects_pb2.Issue.FieldValue(
                  field=resource_name,
                  value=value,
                  derivation=issue_objects_pb2.Issue.Derivation.Value('RULE'))
              for value in values
          ])

    return api_fvs
