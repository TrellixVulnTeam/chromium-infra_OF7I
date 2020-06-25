# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import ast
import json
import logging
import re

from common.findit_http_client import FinditHttpClient
from gae_libs.caches import CompressedMemCache
from gae_libs.gitiles.cached_gitiles_repository import CachedGitilesRepository
from libs import test_name_util
from libs.cache_decorator import Cached
from model.flake.flake import TestLocation
from services import step_util
from services import swarmed_test_util

DEFAULT_VALUE = 'Unknown'

# Url to the file with the mapping from the directories to crbug components.
_COMPONENT_MAPPING_URL = ('https://storage.googleapis.com/chromium-owners/'
                          'component_map_subdirs.json')

# Components for gpu test steps.
# So that Findit can still auto assign the component to some flakes' bugs even
# if cannot get their components based on test location.
_WEBGL_STEP_COMPONENT_LIST = ['Blink>WebGL']
_GPU_STEP_COMPONENT_LIST = ['Internals>GPU>Testing']


@Cached(CompressedMemCache(), expire_time=3600)
def _GetChromiumMapping(mapping_name):
  """Returns a dict mapping."""
  status, content, _ = FinditHttpClient().Get(_COMPONENT_MAPPING_URL)
  if status != 200:
    return {}
  mapping = json.loads(content).get(mapping_name)
  if not mapping:
    return {}
  result = {}
  for path, dict_item in mapping.iteritems():
    path = path + '/' if path[-1] != '/' else path
    result[path] = dict_item
  return result


@Cached(CompressedMemCache(), expire_time=3600)
def _GetChromiumWATCHLISTS():
  repo_url = 'https://chromium.googlesource.com/chromium/src'
  source = CachedGitilesRepository(FinditHttpClient(),
                                   repo_url).GetSource('WATCHLISTS', 'master')
  if not source:
    return None

  # https://cs.chromium.org/chromium/src/WATCHLISTS is in python.
  definitions = ast.literal_eval(source).get('WATCHLIST_DEFINITIONS')
  return dict((k, v['filepath']) for k, v in definitions.iteritems())


def _NormalizePath(path):
  """Returns the normalized path of the given one.

  Normalization include:
  * Convert '\\' to '/'
  * Convert '\\\\' to '/'
  * Resolve '../' and './'

  Example:
  '..\\a/../b/./c/test.cc' --> 'b/c/test.cc'
  """
  path = path.replace('\\', '/')
  path = path.replace('//', '/')

  filtered_parts = []
  for part in path.split('/'):
    if part == '..':
      if filtered_parts:
        filtered_parts.pop()
    elif part == '.':
      continue
    else:
      filtered_parts.append(part)

  return '/'.join(filtered_parts)


def GetChromiumDirectoryToComponentMapping():
  """Returns a dict mapping from directories to components."""
  return _GetChromiumMapping('dir-to-component')


def GetChromiumDirectoryToTeamMapping():
  """Returns a dict mapping from directories to team."""
  return _GetChromiumMapping('dir-to-team')


def GetTestLocation(build_id, step_name, test_name, normalized_test_name):
  """Returns a TestLocation for the given test.

  Currently only supports blink_web_tests and Gtests.

  Args:
    build_id (int): Build id of the build.
    step_name (str): The name of the step.
    test_name (str): The name of the test.
    normalized_step_name (str): The normalized version of the step name.
  """
  if 'webkit_layout_tests' in step_name or 'blink_web_tests' in step_name:
    # For blink_web_tests, assume that the normalized test name is
    # the directory name.
    return TestLocation(
        file_path=_NormalizePath('third_party/blink/web_tests/%s' %
                                 normalized_test_name))
  if not test_name_util.GTEST_REGEX.match(normalized_test_name):
    return None

  # For Gtest, we read the test location from the output.json
  step_metadata = step_util.GetStepMetadata(
      build_id, step_name, partial_match=True)

  task_ids = step_metadata.get('swarm_task_ids') if step_metadata else []
  if not task_ids:
    logging.info(
        "Failed to get step_metadata for  build_id : %s and step_name : %s",
        build_id, step_name)
    return None
  for task_id in task_ids:
    test_path = swarmed_test_util.GetTestLocation(task_id, test_name)
    if test_path:
      return TestLocation(
          file_path=_NormalizePath(test_path.file), line_number=test_path.line)
  return None


def GetTestComponentFromLocation(test_location, component_mapping):
  """Uses test file path to find the best matched component in the mapping.

  Args:
    test_location (TestLocation): The location of a test in the source tree.
    component_mapping (dict): Mapping from directories to crbug components.
  Returns:
    component (str): Test component or 'Unknown' if it could not be found.
  """
  file_path = test_location.file_path
  index = len(file_path)
  while index > 0:
    index = file_path.rfind('/', 0, index)
    if index > 0 and file_path[0:index + 1] in component_mapping:
      return component_mapping[file_path[0:index + 1]]
  return DEFAULT_VALUE


def GetTestDirectoryFromLocation(test_location):
  """Uses test file path to find the directory.

  Args:
    test_location (TestLocation): The location of a test in the source tree.
  Returns:
    directory (str): Test directory
  """
  file_path = test_location.file_path
  index = len(file_path)
  index = file_path.rfind('/', 0, index)
  if index > 0:
    return file_path[0:index + 1]
  return DEFAULT_VALUE


def GetTestTeamFromLocation(test_location, team_mapping):
  """Uses test file path to find the best matched team in the mapping.

  Args:
    test_location (TestLocation): The location of a test in the source tree.
    team_mapping (dict): Mapping from directories to team.
  Returns:
    team (str): Test team or 'Unknown' if it could not be found.
  """
  file_path = test_location.file_path
  index = len(file_path)
  while index > 0:
    index = file_path.rfind('/', 0, index)
    if index > 0 and file_path[0:index + 1] in team_mapping:
      return team_mapping[file_path[0:index + 1]]
  return DEFAULT_VALUE


def GetTagsFromLocation(tags, test_location, component, watchlists):
  """Updates location-based tags for a test.

  Args:
    tags ([str]): Tags that specify the category of the test.
    test_location (TestLocation): The location of a test in the source tree.
    component (str): The component of the test.
    watchlists (dict): Mapping from directories to watchlists.
  Returns:
    Alphabetically sorted tags for a test with updated location-based tags.
  """
  new_tags = set([
      t for t in (tags or [])
      if not t.startswith(('watchlist::', 'directory::', 'source::',
                           'parent_component::', 'component::'))
  ])

  file_path = test_location.file_path
  index = len(file_path)
  while index > 0:
    index = file_path.rfind('/', 0, index)
    if index > 0:
      new_tags.add('directory::%s' % file_path[0:index + 1])
  new_tags.add('source::%s' % file_path)

  for watchlist, pattern in watchlists.iteritems():
    if re.search(pattern, file_path):
      new_tags.add('watchlist::%s' % watchlist)

  new_tags.add('component::%s' % component)
  new_tags.add('parent_component::%s' % component)
  index = len(component)
  while index > 0:
    index = component.rfind('>', 0, index)
    if index > 0:
      new_tags.add('parent_component::%s' % component[0:index])
  return sorted(new_tags)


def GetTestComponentsForGPUTest(build_id, step_name):
  """Use canonical step name to find the best matched components in the mapping.

  Args:
    step_name (str): The name of the step.
    build_id (int): Build id of the build.
  Returns:
    components ([str]): sorted list of components mapped to canonical step name.
  """
  canonical_step_name = step_util.GetCanonicalStepName(
      build_id, step_name) or step_name.split()[0]
  if canonical_step_name.startswith('webgl'):
    return _WEBGL_STEP_COMPONENT_LIST
  return _GPU_STEP_COMPONENT_LIST


def GetTagsForGPUTest(tags, components):
  """Updates component-based tags for a GPU test.

  Args:
    tags ([str]): Tags that specify the category of the test.
    components (set(str)): The components of a test.
  Returns:
    Alphabetically sorted tags for a test with updated component-based tags.
  """
  new_tags = set([
      t for t in (tags or [])
      if not t.startswith(('watchlist::', 'directory::', 'source::',
                           'parent_component::', 'component::'))
  ])
  new_tags = new_tags.union(
      set(['component::%s' % component for component in components]))
  return sorted(new_tags)
