# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

"""Methods for converting resource names to protorpc objects and back.

IngestFoo methods take resource names and return the IDs of the resources.
While some Ingest methods need to check for the existence of resources as
a side-effect of producing their IDs, other layers that call these methods
should always do their own validity checking.

ConvertFoo methods takes protorpc objects and returns resource name.
"""

import re
import logging

from framework import exceptions
from project import project_constants

# Constants that hold regex patterns for resource names.
HOTLIST_PATTERN = 'hotlists\/(?P<hotlist_id>\d+)'
HOTLIST_NAME_RE = re.compile(r'%s$' % HOTLIST_PATTERN)
HOTLIST_ITEM_NAME_RE = re.compile(
    r'%s\/items\/(?P<project_name>%s)\.(?P<local_id>\d+)$' % (
        HOTLIST_PATTERN,
        project_constants.PROJECT_NAME_PATTERN))


def _GetResourceNameMatch(name, regex):
  # (string, Pattern[str]) -> Match[str]
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


def IngestHotlistName(name):
  # string -> int
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
  # List[Tuple[project_name, issue_local_id]] -> Collection(ints)
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
