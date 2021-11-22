# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""A script to aggregate files coverage data to directories and components.

The code coverage data format is defined at:
https://chromium.googlesource.com/infra/infra/+/refs/heads/main/appengine/findit/model/proto/code_coverage.proto

"""

from collections import defaultdict
import logging
import os
import posixpath


def _new_summaries(metrics):
  """Returns new summaries for the input metrics."""
  return [{'name': metric, 'covered': 0, 'total': 0} for metric in metrics]


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


class SummaryCoverageAggregator(object):

  def __init__(self, metrics):
    self.per_directory_summaries = defaultdict(lambda: _new_summaries(metrics))
    self.per_directory_files = defaultdict(list)

  def _update_per_directory_summaries(self, file_coverage):
    parent_dir = posixpath.dirname(file_coverage['path'])
    while parent_dir != '//':
      # In the coverage data format, dirs end with '/' except for root.
      parent_coverage_path = parent_dir + '/'
      _merge_summary(self.per_directory_summaries[parent_coverage_path],
                     file_coverage['summaries'])
      parent_dir = posixpath.dirname(parent_dir)

    _merge_summary(self.per_directory_summaries['//'],
                   file_coverage['summaries'])

  def _update_per_directory_files(self, file_coverage):
    direct_parent_dir = posixpath.dirname(file_coverage['path'])
    if direct_parent_dir != '//':
      # In the coverage data format, dirs end with '/' except for root.
      direct_parent_dir += '/'

    self.per_directory_files[direct_parent_dir].append({
        'name': posixpath.basename(file_coverage['path']),
        'path': file_coverage['path'],
        'summaries': file_coverage['summaries'],
    })

  def _calculate_per_directory_subdirs(self):
    """Calculates and returns per directory sub directories CoverageSummary.

    Returns:
      A dict mapping from directory to a list of CoverageSummary for directory.
    """
    per_directory_subdirs = defaultdict(list)
    for dir_path in sorted(self.per_directory_summaries.keys()):
      if dir_path == '//':
        continue
      assert dir_path.endswith('/'), (
          'Directory path: %s is expected to end with / in coverage data format'
          % dir_path)
      parent_dir_path, dirname = posixpath.split(dir_path[:-1])
      if parent_dir_path != '//':
        parent_dir_path += '/'
      per_directory_subdirs[parent_dir_path].append({
          'name': dirname + '/',
          'path': dir_path,
          'summaries': self.per_directory_summaries[dir_path],
      })
    return per_directory_subdirs

  def consume_file_coverage(self, file_coverage):
    """Consumes coverage data for a single file.
    
    Processes the coverage data for a single file and updates internal
    aggregated metrics.s
    """
    self._update_per_directory_summaries(file_coverage)
    self._update_per_directory_files(file_coverage)

  def produce_summary_coverage(self):
    """Returns aggregated coverage metrics.
    
    Should ideally be called at the end of `consume_file_coverage()` calls."""
    per_directory_subdirs = self._calculate_per_directory_subdirs()
    per_directory_coverage_data = {}
    for dir_path in self.per_directory_summaries:
      per_directory_coverage_data[dir_path] = {
          'path': dir_path,
          'dirs': per_directory_subdirs[dir_path],
          'files': self.per_directory_files[dir_path],
          'summaries': self.per_directory_summaries[dir_path],
      }
    return per_directory_coverage_data
