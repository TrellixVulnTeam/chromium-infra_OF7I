# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.from datetime import datetime

import cloudstorage

from common.findit_http_client import FinditHttpClient
from gae_libs.gitiles.cached_gitiles_repository import CachedGitilesRepository
from model.code_coverage import DependencyRepository
from waterfall import waterfall_config

# Cloud storage bucket used to store the source files fetched from gitile.
_SOURCE_FILE_GS_BUCKET = 'source-files-for-coverage'


def GetPostsubmitPlatformInfoMap(luci_project):
  """Returns a map of postsubmit platform information.

  The map contains per-luci_project platform information, and following is
  an example config:
  {
    'postsubmit_platform_info_map': {
      'chromium': {
        'linux': {
          'bucket': 'ci',
          'buider': 'linux-code-coverage',
          'coverage_tool': 'clang',
          'ui_name': 'Linux (C/C++)',
        }
      }
    }
  }
  """
  return waterfall_config.GetCodeCoverageSettings().get(
      'postsubmit_platform_info_map', {}).get(luci_project, {})


def _GetAllowedGitilesConfigs():
  """Returns the set of valid gitiles configurations.

  The returned structure contains the tree of valid hosts, projects, and refs.

  Please note that the hosts in the config are gitiles hosts instead of gerrit
  hosts, such as: 'chromium.googlesource.com'.

  Example config:
  {
    'allowed_gitiles_configs': {
      'chromium.googlesource.com': {
        'chromium/src': [
          'refs/heads/main',
        ]
      }
    }
  }
  """
  return waterfall_config.GetCodeCoverageSettings().get(
      'allowed_gitiles_configs', {})


def _GetMatchedDependencyRepository(manifest, file_path):  # pragma: no cover.
  """Gets the matched dependency in the manifest of the report.

  Args:
    manifest (DependencyRepository): Entity containing mapping from path prefix
                                     to corresponding repo.
    file_path (str): Source absolute path to the file.

  Returns:
    A DependencyRepository if a matched one is found and it is allowed,
    otherwise None.
  """
  assert file_path.startswith('//'), 'All file path should start with "//".'

  for dep in manifest:
    if file_path.startswith(
        dep.path) and dep.server_host in _GetAllowedGitilesConfigs():
      return dep

  return None


def GetFileContentFromGitiles(manifest, file_path,
                              revision):  # pragma: no cover.
  """Fetches the content of a specific revision of a file from gitiles.

  Args:
    manifest (DependencyRepository): Entity containing mapping from path prefix
                                     to corresponding repo.
    file_path (str): Source absolute path to the file.
    revision (str): The gitile revision of the file.

  Returns:
    The content of the source file."""
  assert file_path.startswith('//'), 'All file path should start with "//".'
  assert revision, 'A valid revision is required'

  dependency = _GetMatchedDependencyRepository(manifest, file_path)
  assert dependency, ('%s file does not belong to any dependency repository' %
                      file_path)

  # Calculate the relative path to the root of the dependency repository itself.
  relative_file_path = file_path[len(dependency.path):]
  repo = CachedGitilesRepository(FinditHttpClient(), dependency.project_url)
  return repo.GetSource(relative_file_path, revision)


def ComposeSourceFileGsPath(manifest, file_path, revision):
  """Composes a cloud storage path for a specific revision of a source file.

  Args:
    manifest (DependencyRepository): Entity containing mapping from path prefix
                                     to corresponding repo.
    file_path (str): Source absolute path to the file.
    revision (str): The gitile revision of the file in its own repo.

  Returns:
    Cloud storage path to the file, in the format /bucket/object. For example,
    /source-files-for-coverage/chromium.googlesource.com/v8/v8/src/date.cc/1234.
  """
  assert file_path.startswith('//'), 'All file path should start with "//".'
  assert revision, 'A valid revision is required'

  dependency = _GetMatchedDependencyRepository(manifest, file_path)
  assert dependency, ('%s file does not belong to any dependency repository' %
                      file_path)

  # Calculate the relative path to the root of the dependency repository itself.
  relative_file_path = file_path[len(dependency.path):]
  return '/%s/%s/%s/%s/%s' % (_SOURCE_FILE_GS_BUCKET, dependency.server_host,
                              dependency.project, relative_file_path, revision)


def WriteFileContentToGs(gs_path, content):  # pragma: no cover.
  """Writes the content of a file to cloud storage.

  Args:
    gs_path (str): Path to the file, in the format /bucket/object.
    content (str): Content of the file.
  """
  write_retry_params = cloudstorage.RetryParams(backoff_factor=2)
  with cloudstorage.open(
      gs_path, 'w', content_type='text/plain',
      retry_params=write_retry_params) as f:
    f.write(content)
