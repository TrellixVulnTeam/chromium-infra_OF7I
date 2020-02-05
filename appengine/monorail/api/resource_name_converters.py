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
