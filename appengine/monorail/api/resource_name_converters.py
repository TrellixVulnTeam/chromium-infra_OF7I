# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

"""Methods for converting resource names to protorpc objects and back.

IngestFoo methods take resource names and return the IDs of the resources.
While some Ingest methods need to check for the existence of resources as
a side-effect of producing their IDs, other layers that call these methods
should always do their own validity checking.

ConvertFoo methods take object ids
(and sometimes a MonorailConnection and ServiceManager)
and return resource names.
"""

import re
import logging

from framework import exceptions
from project import project_constants

# Constants that hold regex patterns for resource names.
PROJECT_NAME_PATTERN = (
    r'projects\/(?P<project_name>%s)' % project_constants.PROJECT_NAME_PATTERN)
PROJECT_NAME_RE = re.compile(r'%s$' % PROJECT_NAME_PATTERN)

HOTLIST_PATTERN = r'hotlists\/(?P<hotlist_id>\d+)'
HOTLIST_NAME_RE = re.compile(r'%s$' % HOTLIST_PATTERN)
HOTLIST_ITEM_NAME_RE = re.compile(
    r'%s\/items\/(?P<project_name>%s)\.(?P<local_id>\d+)$' % (
        HOTLIST_PATTERN,
        project_constants.PROJECT_NAME_PATTERN))

ISSUE_PATTERN = (r'projects\/(?P<project>%s)\/issues\/(?P<local_id>\d+)' %
                 project_constants.PROJECT_NAME_PATTERN)
ISSUE_NAME_RE = re.compile(r'%s$' % ISSUE_PATTERN)

USER_NAME_RE = re.compile(r'users\/(?P<user_id>\d+)$')

# Constants that hold the template patterns for creating resource names.
HOTLIST_NAME_TMPL = 'hotlists/{hotlist_id}'
HOTLIST_ITEM_NAME_TMPL = '%s/items/{project_name}.{local_id}' % (
    HOTLIST_NAME_TMPL)

ISSUE_NAME_TMPL = 'projects/{project}/issues/{local_id}'

USER_NAME_TMPL = 'users/{user_id}'

ISSUE_TEMPLATE_TMPL = 'projects/{project_name}/templates/{template_name}'
STATUS_DEF_TMPL = 'projects/{project_name}/statusDefs/{status}'
LABEL_DEF_TMPL = 'projects/{project_name}/labelDefs/{label}'
COMPONENT_DEF_TMPL = 'projects/{project_name}/componentDefs/{component_id}'
FIELD_DEF_TMPL = 'projects/{project_name}/fieldDefs/{field_name}'
APPROVAL_DEF_TMPL = 'projectConfigs/{project_name}/approvalDefs/{approval_name}'


def _GetResourceNameMatch(name, regex):
  # (str, Pattern[str]) -> Match[str]
  """Takes a resource name and returns the regex match.

  Args:
    name: Resource name.
    regex: Compiled regular expression Pattern object used to match name.

  Raises:
    InputException if there is not match.
  """
  match = regex.match(name)
  if not match:
    raise exceptions.InputException(
        'Invalid resource name: %s.' % name)
  return match


def _IssueIdsFromLocalIds(cnxn, project_local_id_pairs, services):
  # (MonorailConnection, Sequence[Tuple(str, int)], Services -> Sequence[int]
  """Fetches issue IDs using the given project/local ID pairs."""
  # Fetch Project ids from Project names.
  project_ids_by_name = services.project.LookupProjectIDs(
      cnxn, [pair[0] for pair in project_local_id_pairs])

  # Create (project_id, issue_local_id) pairs from project_local_id_pairs.
  project_id_local_ids = []
  for project_name, local_id in project_local_id_pairs:
    try:
      project_id = project_ids_by_name[project_name]
      project_id_local_ids.append((project_id, local_id))
    except KeyError:
      raise exceptions.NoSuchProjectException(
          'Project %s not found' % project_name)

  issue_ids, misses = services.issue.LookupIssueIDs(cnxn, project_id_local_ids)
  if misses:
    raise exceptions.NoSuchIssueException('Issue(s) %r not found' % misses)
  return issue_ids

# Hotlists

def IngestHotlistName(name):
  # str -> int
  """Takes a Hotlist resource name and returns the Hotlist ID.

  Args:
    name: Resource name of a Hotlist.

  Returns:
    The Hotlist's ID

  Raises:
    InputException if the given name does not have a valid format.
  """
  match = _GetResourceNameMatch(name, HOTLIST_NAME_RE)
  return int(match.group('hotlist_id'))


def IngestHotlistItemNames(cnxn, names, services):
  # (MonorailConnection, Sequence[str], Services -> Sequence[int]
  """Takes HotlistItem resource names and returns the associated Issues' IDs.

  Args:
    cnxn: MonorailConnection to the database.
    names: List of HotlistItem resource names.
    services: Services object for connections to backend services.

  Returns:
    List of Issue IDs associated with the given HotlistItems.

  Raises:
    InputException if a resource name does not have a valid format.
    NoSuchProjectException if an Issue's Project is not found.
    NoSuchIssueException if an Issue is not found.
  """
  project_local_id_pairs = []
  for name in names:
    match = _GetResourceNameMatch(name, HOTLIST_ITEM_NAME_RE)
    project_local_id_pairs.append(
        (match.group('project_name'), int(match.group('local_id'))))
  return _IssueIdsFromLocalIds(cnxn, project_local_id_pairs, services)



def ConvertHotlistName(hotlist_id):
  # int -> str
  """Takes a Hotlist and returns the Hotlist's resource name.

  Args:
    hotlist_id: ID of the Hotlist.

  Returns:
    The resource name of the Hotlist.
  """
  return HOTLIST_NAME_TMPL.format(hotlist_id=hotlist_id)


def ConvertHotlistItemNames(cnxn, hotlist_id, issue_ids, services):
  # MonorailConnection, int, Collection[int], Services -> Mapping[int, str]
  """Takes a Hotlist ID and HotlistItem's issue_ids and returns
     the Hotlist items' resource names.

  Args:
    cnxn: MonorailConnection object.
    hotlist_id: ID of the Hotlist the items belong to.
    issue_ids: List of Issue IDs that are part of the hotlist's items.
    services: Services object for connections to backend services.

  Returns:
    Dict of Issue IDs to HotlistItem resource names for Issues that are found.
  """
  # {issue_id: (project_name, local_id),...}
  issue_refs_dict = services.issue.LookupIssueRefs(cnxn, issue_ids)

  issue_ids_to_names = {}
  for issue_id in issue_ids:
    project_name, local_id = issue_refs_dict.get(issue_id, (None, None))
    if project_name and local_id:
      issue_ids_to_names[issue_id] = HOTLIST_ITEM_NAME_TMPL.format(
          hotlist_id=hotlist_id, project_name=project_name, local_id=local_id)

  return issue_ids_to_names

# Issues


def IngestIssueName(cnxn, name, services):
  # MonorailConnection, str, Services -> int
  """Takes an Issue resource name and returns its global ID.

  Args:
    cnxn: MonorailConnection object.
    name: Resource name of an Issue.
    services: Services object for connections to backend services.

  Returns:
    The global Issue ID associated with the name.

  Raises:
    InputException if the given name does not have a valid format.
    NoSuchIssueException if the Issue does not exist.
    NoSuchProjectException if an Issue's Project is not found.

  """
  return IngestIssueNames(cnxn, [name], services)[0]


def IngestIssueNames(cnxn, names, services):
  # MonorailConnection, Sequence[str], Services -> Sequence[int]
  """Returns global IDs for the given Issue resource names.

  Args:
    cnxn: MonorailConnection object.
    names: Resource names of zero or more issues.
    services: Services object for connections to backend services.

  Returns:
    The global IDs for the issues.

  Raises:
    InputException if a resource name does not have a valid format.
    NoSuchIssueException if an Issue is not found.
    NoSuchProjectException if an Issue's Project is not found.
  """
  project_local_id_pairs = []
  for name in names:
    match = _GetResourceNameMatch(name, ISSUE_NAME_RE)
    project_local_id_pairs.append(
        (match.group('project'), int(match.group('local_id'))))
  return _IssueIdsFromLocalIds(cnxn, project_local_id_pairs, services)



def ConvertIssueName(cnxn, issue_id, services):
  # MonorailConnection, int, Service -> str
  """Takes an Issue ID and returns the corresponding Issue resource name.

  Args:
    mc: MonorailConnection object.
    issue_id: The ID of the issue.
    services: Service object.

  Returns:
    The resource name of the Issue.

  Raises:
    NoSuchIssueException if the issue is not found.
  """
  name = ConvertIssueNames(cnxn, [issue_id], services).get(issue_id)
  if not name:
    raise exceptions.NoSuchIssueException()
  return name


def ConvertIssueNames(cnxn, issue_ids, services):
  # MonorailConnection, Collection[int], Service -> Mapping[int, str]
  """Takes Issue IDs and returns the Issue resource names.

  Args:
    cnxn: MonorailConnection object.
    issue_ids: List of Issue IDs
    services: Service object.

  Returns:
    Dict of Issue IDs to Issue resource names for Issues that are found.
  """
  issue_ids_to_names = {}
  issue_refs_dict = services.issue.LookupIssueRefs(cnxn, issue_ids)
  for issue_id in issue_ids:
    project, local_id = issue_refs_dict.get(issue_id, (None, None))
    if project and local_id:
      issue_ids_to_names[issue_id] = ISSUE_NAME_TMPL.format(
          project=project, local_id=local_id)
  return issue_ids_to_names


# Users

def IngestUserNames(names):
  # Sequence[str] -> Sequence[int]
  """Takes a User resource names and returns the User IDs.

  Args:
    names: List of User resource names.

  Returns:
    List of User IDs in the same order as names.

  Raises:
    InputException if an resource name does not have a valid format.
  """
  ids = []
  for name in names:
    match = _GetResourceNameMatch(name, USER_NAME_RE)
    ids.append(int(match.group('user_id')))

  return ids


def ConvertUserNames(user_ids):
  # Collection[int] -> Mapping[int, str]
  """Takes a List of Users and returns the User's resource names.

  Args:
    user_ids: List of User IDs.

  Returns:
    Dict of User IDs to User resource names for all given user_ids.
  """
  user_ids_to_names = {}
  for user_id in user_ids:
    user_ids_to_names[user_id] = USER_NAME_TMPL.format(user_id=user_id)

  return user_ids_to_names


# Projects


def IngestProjectName(cnxn, name, services):
  # str -> int
  """Takes a Project resource name and returns the project id.

  Args:
    name: Resource name of a Project.

  Returns:
    The project's id

  Raises:
    InputException if the given name does not have a valid format.
  """
  match = _GetResourceNameMatch(name, PROJECT_NAME_RE)
  project_name = match.group('project_name')

  id_dict = services.project.LookupProjectIDs(cnxn, [project_name])

  return id_dict.get(project_name)


def ConvertTemplateNames(cnxn, project_id, template_ids, services):
  # MonorailConnection, int, Collection[int] Service -> Mapping[int, str]
  """Takes Template IDs and returns the Templates' resource names

  Args:
    cnxn: MonorailConnection object.
    project_id: project id of project that templates belong to
    template_ids: list of template ids
    services: Service object.

  Returns:
    Dict of template ID to template resource names for all found template ids.

  Raises:
    NoSuchProjectException if no project exists with given id.
  """
  id_to_resource_names = {}

  project_name = services.project.LookupProjectNames(
      cnxn, [project_id]).get(project_id)
  if project_name is None:
    raise exceptions.NoSuchProjectException(project_id)
  templates = services.template.GetTemplatesById(cnxn, template_ids)

  for template in templates:
    id_to_resource_names[template.template_id] = ISSUE_TEMPLATE_TMPL.format(
        project_name=project_name, template_name=template.name)

  return id_to_resource_names


def ConvertStatusDefName(cnxn, status, project_id, services):
  # MonorailConnection, str, int, Service -> str
  """Takes a status string and returns a StatusDef resource name

  Args:
    cnxn: MonorailConnection object.
    status: status name as string
    project_id: project id of project this belongs to
    services: Service object.

  Returns:
    String of status's resource name

  Raises:
    NoSuchProjectException if no project exists with given id.
  """
  project = services.project.GetProject(cnxn, project_id)

  return STATUS_DEF_TMPL.format(
      project_name=project.project_name, status=status)


def ConvertLabelDefNames(cnxn, labels, project_id, services):
  # MonorailConnection, Collection[str], int, Service -> Mapping[str, str]
  """Takes a list of labels and returns LabelDef resource names

  Args:
    cnxn: MonorailConnection object.
    labels: List of labels as string
    project_id: project id of project this belongs to
    services: Service object.

  Returns:
    Dict of label string to label's resource name for all given `labels`.

  Raises:
    NoSuchProjectException if no project exists with given id.
  """
  project = services.project.GetProject(cnxn, project_id)

  name_dict = {}

  for label in labels:
    name_dict[label] = LABEL_DEF_TMPL.format(
        project_name=project.project_name, label=label)

  return name_dict


def ConvertComponentDefNames(cnxn, component_ids, project_id, services):
  # MonorailConnection, Collection[int], int, Service -> Mapping[int, str]
  """Takes Component IDs and returns ComponentDef resource names

  Args:
    cnxn: MonorailConnection object.
    component_ids: List of component ids
    project_id: project id of project this belongs to
    services: Service object.

  Returns:
    Dict of component ID to component's resource name for all given
    `component_ids`

  Raises:
    NoSuchProjectException if no project exists with given id.
  """
  project = services.project.GetProject(cnxn, project_id)

  id_dict = {}

  for component_id in component_ids:
    id_dict[component_id] = COMPONENT_DEF_TMPL.format(
        project_name=project.project_name, component_id=component_id)

  return id_dict


def ConvertFieldDefNames(cnxn, field_ids, project_id, services):
  # MonorailConnection, Collection[int], int, Service -> Mapping[int, str]
  """Takes Field IDs and returns FieldDef resource names

  Args:
    cnxn: MonorailConnection object.
    component_ids: List of field ids
    project_id: project id of project this belongs to
    services: Service object.

  Returns:
    Dict of field ID to FieldDef resource name for field defs that are found.

  Raises:
    NoSuchProjectException if no project exists with given id.
  """
  project = services.project.GetProject(cnxn, project_id)
  config = services.config.GetProjectConfig(cnxn, project_id)

  fds_by_id = {fd.field_id: fd for fd in config.field_defs}

  id_dict = {}

  for field_id in field_ids:
    field_def = fds_by_id.get(field_id)
    if not field_def:
      logging.info('Ignoring field referencing a non-existent id: %s', field_id)
      continue
    field_name = field_def.field_name

    id_dict[field_id] = FIELD_DEF_TMPL.format(
        project_name=project.project_name, field_name=field_name)

  return id_dict


def ConvertApprovalDefNames(cnxn, approval_ids, project_id, services):
  # type: (MonorailConnection, Collection[int], int, Services) ->
  #     Mapping[int, str]
  """Takes Approval IDs and returns ApprovalDef resource names

  Args:
    cnxn: MonorailConnection object.
    component_ids: List of approval ids
    project_id: project id of project this belongs to
    services: Services object.

  Returns:
    Dict of approval ID to ApprovalDef resource name for approval defs
    that are found.

  Raises:
    NoSuchProjectException if no project exists with given id.
  """
  project = services.project.GetProject(cnxn, project_id)
  config = services.config.GetProjectConfig(cnxn, project_id)

  fds_by_id = {fd.field_id: fd for fd in config.field_defs}

  id_dict = {}

  for approval_id in approval_ids:
    approval_def = fds_by_id.get(approval_id)
    if not approval_def:
      logging.info(
          'Ignoring approval referencing a non-existent id: %s', approval_id)
      continue
    approval_name = approval_def.field_name

    id_dict[approval_id] = APPROVAL_DEF_TMPL.format(
        project_name=project.project_name, approval_name=approval_name)

  return id_dict
