# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""A script to aggregate files coverage data to directories and components.

The code coverage data format is defined at:
https://chromium.googlesource.com/infra/infra/+/refs/heads/main/appengine/findit/model/proto/code_coverage.proto

NOTE: This is a copy of code at https://bit.ly/3FI39O0 and should be kept in
sync till crbug.com/1259714 is resolved.
"""

from collections import defaultdict
import os
import posixpath


def get_aggregated_coverage_data_from_files(files_coverage_data,
                                            dir_to_component=None):
  """Aggregates files coverage data to directories and components.

  Note: this function assumes that all files are written in the same language,
  and have same metrics

  Args:
    files_coverage_data (list): A list of File coverage data.
    dir_to_component (dict): Mapping from directory to component. Optional.

  Returns:
    A tuple of two elements: the first one is a dict mapping from directory to
    GroupCoverageSummary data, and the other one is a dict mapping from
    component to GroupCoverageSummary data.
  """
  per_directory_summaries = _caclulate_per_directory_summaries(
      files_coverage_data)
  per_directory_files = _calculate_per_directory_files(files_coverage_data)
  per_directory_subdirs = _calculate_per_directory_subdirs(
      per_directory_summaries)
  per_directory_coverage_data = {}
  for dir_path in per_directory_summaries:
    per_directory_coverage_data[dir_path] = {
        'path': dir_path,
        'dirs': per_directory_subdirs[dir_path],
        'files': per_directory_files[dir_path],
        'summaries': per_directory_summaries[dir_path],
    }

  if not dir_to_component:
    return per_directory_coverage_data, None

  component_to_dirs = _extract_component_to_dirs_mapping(dir_to_component)
  per_component_coverage_data = {}
  for component in component_to_dirs:
    summaries = _new_summaries(per_directory_summaries['//'])
    for dir_path in component_to_dirs[component]:
      _merge_summary(summaries, per_directory_summaries[dir_path])

    sub_dirs = [{
        'path': dir_path,
        'name': posixpath.basename(dir_path[:-1]) + '/',
        'summaries': per_directory_summaries[dir_path],
    } for dir_path in component_to_dirs[component]]

    per_component_coverage_data[component] = {
        'path': component,
        'dirs': sub_dirs,
        'summaries': summaries,
    }

  return per_directory_coverage_data, per_component_coverage_data


def _caclulate_per_directory_summaries(files_coverage_data):
  """Calculates and returns per directory coverage metric summaries.

  Args:
    files_coverage_data (list): A list of File coverage data.

  Returns:
    A dict mapping from directory to coverage metric summaries.
  """
  per_directory_summaries = defaultdict(lambda: _new_summaries(
      files_coverage_data[0]['summaries']))
  for file_record in files_coverage_data:
    parent_dir = posixpath.dirname(file_record['path'])
    while parent_dir != '//':
      # In the coverage data format, dirs end with '/' except for root.
      parent_coverage_path = parent_dir + '/'
      _merge_summary(per_directory_summaries[parent_coverage_path],
                     file_record['summaries'])
      parent_dir = posixpath.dirname(parent_dir)

    _merge_summary(per_directory_summaries['//'], file_record['summaries'])

  return per_directory_summaries


def _calculate_per_directory_files(files_coverage_data):
  """Calculates and returns per directory files CoverageSummary.

  Args:
    files_coverage_data (list): A list of File coverage data.

  Returns:
    A dict mapping from directory to a list of CoverageSummary for file.
  """
  per_directory_files = defaultdict(list)
  for file_record in files_coverage_data:
    direct_parent_dir = posixpath.dirname(file_record['path'])
    if direct_parent_dir != '//':
      # In the coverage data format, dirs end with '/' except for root.
      direct_parent_dir += '/'

    per_directory_files[direct_parent_dir].append({
        'name': posixpath.basename(file_record['path']),
        'path': file_record['path'],
        'summaries': file_record['summaries'],
    })

  return per_directory_files


def _calculate_per_directory_subdirs(per_directory_summaries):
  """Calculates and returns per directory sub directories CoverageSummary.

  Args:
    per_directory_summaries (dict): A dict mapping from directory to coverage
                                    metric summaries.

  Returns:
    A dict mapping from directory to a list of CoverageSummary for directory.
  """
  per_directory_subdirs = defaultdict(list)
  for dir_path in sorted(per_directory_summaries.keys()):
    if dir_path == '//':
      continue

    assert dir_path.endswith('/'), (
        'Directory path: %s is expected to end with / in coverage data format' %
        dir_path)
    parent_dir_path, dirname = posixpath.split(dir_path[:-1])
    if parent_dir_path != '//':
      parent_dir_path += '/'

    per_directory_subdirs[parent_dir_path].append({
        'name': dirname + '/',
        'path': dir_path,
        'summaries': per_directory_summaries[dir_path],
    })

  return per_directory_subdirs


def _extract_component_to_dirs_mapping(dir_to_component):
  """Extracts mapping from component to directories.

  Note that this method avoids double-counting, meaning that if there are two
  directories, where one if the ancestor of the other one, mapping to the same
  component, then only the ancestor directory is present in the output.

  For example:
    dir_to_component: {'dir': 'Test>Component', 'dir/subdir': 'Test>Component'}
    component_to_dirs: {'Test>Component': 'dir'} # No subdir.

  Args:
    dir_to_component (dict): Mapping from directory to component.

  Returns:
    A dict mapping from component to a list of directories.
  """
  component_to_dirs = defaultdict(list)

  # Paths in dir to component mapping is relative path that always uses '/' as
  # path separator and does NOT end with '/', for example, 'media/cast'.
  for dir_path, component in sorted(dir_to_component.iteritems()):

    # Check if we already added the parent directory of this directory. If
    # yes, skip this sub-directory to avoid double-counting.
    found_parent_same_component = False
    parent_dir_path = posixpath.dirname(dir_path)
    while parent_dir_path:
      if dir_to_component.get(parent_dir_path) == component:
        found_parent_same_component = True
        break

      parent_dir_path = posixpath.dirname(parent_dir_path)

    if found_parent_same_component:
      continue

    component_to_dirs[component].append('//' + dir_path + '/')

  return component_to_dirs


def _merge_summary(merge_dest, merge_src):
  """Merges to 'summaries' fields in metadata format.

  Two sumaries are required to have the exact same metrics, and this method
  adds the 'total' and 'covered' field of each metric in the second parameter
  to the corresponding field in the first parameter.

  Each parameter is expected to be in the following format:
  [{'name': 'line', 'total': 10, 'covered': 9},
   {'name': 'region', 'total': 10, 'covered': 9},
   {'name': 'function', 'total': 10, 'covered': 9}]
  """

  def get_metrics(summaries):
    return {s['name'] for s in summaries}

  assert get_metrics(merge_dest) == get_metrics(merge_src), (
      '%s and %s are expected to have the same metrics' %
      (merge_dest, merge_src))

  merge_src_dict = {i['name']: i for i in merge_src}
  for merge_dest_item in merge_dest:
    for field in ('total', 'covered'):
      merge_dest_item[field] += merge_src_dict[merge_dest_item['name']][field]


def _new_summaries(reference_summaries):
  """Returns new summaries with the same metrics as the reference one."""
  return [{
      'name': summary['name'],
      'covered': 0,
      'total': 0
  } for summary in reference_summaries]
