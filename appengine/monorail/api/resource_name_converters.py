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
  project_names_local_id = []
  for name in names:
    match = _GetResourceNameMatch(name, HOTLIST_ITEM_NAME_RE)
    project_names_local_id.append(
        (match.group('project_name'), int(match.group('local_id'))))

  # Fetch Project ids from Project names.
  project_ids_by_name = services.project.LookupProjectIDs(
      cnxn, [pair[0] for pair in project_names_local_id])

  # Create (project_id, issue_local_id) pairs from project_names_local_id
  project_id_local_ids = []
  for project_name, local_id in project_names_local_id:
    try:
      project_id = project_ids_by_name[project_name]
      project_id_local_ids.append((project_id, local_id))
    except KeyError:
      raise exceptions.NoSuchProjectException(
          'project %s not found' % project_name)

  issue_ids, misses = services.issue.LookupIssueIDs(
      cnxn, project_id_local_ids)
  if misses:
    raise exceptions.NoSuchIssueException(
        'Issue(s) %r associated with HotlistItems not found' % misses)
  return issue_ids

def ConvertHotlistName(hotlist_id):
  # int -> str
  """Takes a Hotlist and returns the Hotlist's resource name.

  Args:
    hotlist_id: ID of the Hotlist.

  Returns:
    The resource name of the Hotlist.
  """
  return HOTLIST_NAME_TMPL.format(hotlist_id=hotlist_id)


def ConvertHotlistItemNames(cnxn, hotlist_id, services):
  # MonorailConnection, Hotlist, Services -> Sequence[str]
  """Takes a Hotlist and returns the Hotlist items' resource names.

  Args:
    cnxn: MonorailConnection object.
    hotlist_id: ID of the Hotlist.
    services: Services object for connections to backend services.

  Returns:
    List of resource names of the Hotlist's items in the same order
      that they appear in the Hotlist.

  Raises:
    NoSuchHotlistException if a Hotlist with the hotlist_id does not exist.
  """
  hotlist = services.features.GetHotlist(cnxn, hotlist_id)
  issue_ids = [item.issue_id for item in hotlist.items]

  # {issue_id: (project_name, local_id),...}
  issue_refs_dict = services.issue.LookupIssueRefs(cnxn, issue_ids)

  names = []
  for issue_id in issue_ids:
    project_name, local_id = issue_refs_dict.get(issue_id, (None, None))
    if project_name and local_id:
      names.append(
          HOTLIST_ITEM_NAME_TMPL.format(
              hotlist_id=hotlist.hotlist_id,
              project_name=project_name, local_id=local_id))

  return names


# Issues

def IngestIssueName(name):
  # str -> int
  """Takes an Issue resource name and returns its global ID.

  Args:
    name: Resource name of an Issue.

  Returns:
    The global Issue ID associated with the name.

  Raises:
    InputException if the given name does not have a valid format.
  """
  _GetResourceNameMatch(name, ISSUE_NAME_RE)
  raise Exception('Not implemented')


def ConvertIssueName(project, local_id):
  # str, int -> str
  """Takes a project and local_id returns the corresponding Issue resource name.

  Args:
    project: The string representing the project of the Issue.
    local_id: The local_id of the issue.

  Returns:
    The resource name of the Issue.
  """
  return ISSUE_NAME_TMPL.format(project=project, local_id=local_id)

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
  # Sequence[int] -> Sequence[str]
  """Takes a List of Users and returns the User's resource names.

  Args:
    user_ids: List of User IDs.

  Returns:
    List of user resource names in the same order as user_ids.
  """
  names = []
  for user_id in user_ids:
    names.append(USER_NAME_TMPL.format(user_id=user_id))

  return names
