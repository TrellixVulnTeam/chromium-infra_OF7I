# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""This module is to process the code coverage metadata.

The code coverage data format is defined at:
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/appengine/findit/model/proto/code_coverage.proto.
"""

import collections
import json
import logging
import re
import time
import urlparse
import zlib

import cloudstorage

from google.appengine.api import taskqueue
from google.appengine.api import users
from google.appengine.ext import ndb
from google.protobuf.field_mask_pb2 import FieldMask
from google.protobuf import json_format

from common import monitoring
from common.findit_http_client import FinditHttpClient
from common.waterfall.buildbucket_client import GetV2Build
from gae_libs.appengine_util import IsInternalInstance
from gae_libs.caches import PickledMemCache
from gae_libs.dashboard_util import GetPagedResults
from gae_libs.handlers.base_handler import BaseHandler, Permission
from gae_libs.gitiles.cached_gitiles_repository import CachedGitilesRepository
from libs.cache_decorator import Cached
from libs.deps import chrome_dependency_fetcher
from libs.time_util import ConvertUTCToPST
from model import entity_util
from model.proto.gen.code_coverage_pb2 import CoverageReport
from model.code_coverage import DependencyRepository
from model.code_coverage import PostsubmitReport
from model.code_coverage import FileCoverageData
from model.code_coverage import PresubmitCoverageData
from model.code_coverage import SummaryCoverageData
from services.code_coverage import code_coverage_util
from waterfall import waterfall_config

# The regex to extract the build id from the url path.
_BUILD_ID_REGEX = re.compile(r'.*/build/(\d+)$')

# Cloud storage bucket used to store the source files fetched from gitile.
_SOURCE_FILE_GS_BUCKET = 'source-files-for-coverage'

# The regex to extract the luci project name from the url path.
_LUCI_PROJECT_REGEX = re.compile(r'^/p/(.+)/coverage.*')


def _GetAllowedGitilesHosts():
  """Returns a set of gitiles hosts that the service supports.

  Example config:
  {
    'allowed_gitiles_hosts': ['chromium.googlesource.com'],
  }
  """
  return set(waterfall_config.GetCodeCoverageSettings().get(
      'allowed_gitiles_hosts', []))


def _GetAllowedProjectsForPerCL(host):
  """Returns a set of Gerrit projects that the service supports.

  The main use case is to filter out per-cl coverage data requests whose
  projects are not supported yet to avoid wasting time and resource quering the
  datastore.

  Please note that the hosts in the config are gitiles hosts instead of gerrit
  hosts, such as: 'chromium.googlesource.com'.

  Example config:
  {
    'allowed_projects_for_per_cl': {
        'chromium.googlesource.com': ['chromium/src'],
    },
  }
  """
  return set(waterfall_config.GetCodeCoverageSettings().get(
      'allowed_projects_for_per_cl', {}).get(host, []))


def _GetBlacklistedDeps():
  """Returns a map of blacklisted dependencies to skip adding to manifest.

  Main use case is to skip dependency repos that have malformed structures, and
  the mapping is from root repo url to list of dependency paths (relative to
  the root of the checkout).
  """
  return waterfall_config.GetCodeCoverageSettings().get('blacklisted_deps', {})


def _GetPostsubmitPlatformInfoMap(luci_project):
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


def _GetPostsubmitDefaultReportConfig(luci_project):
  """Returns a tuple of (host, project, ref, platform) to serve default report.

  Following is an example config:
  {
    'default_postsubmit_report_config': {
      'chromium': {
        'host': 'chromium.googlesource.com',
        'project': 'chromium/src',
        'ref': 'refs/heads/master',
        'platform': 'linux',
      }
    }
  }
  """
  return waterfall_config.GetCodeCoverageSettings().get(
      'default_postsubmit_report_config', {}).get(luci_project, None)


def _GetWhitelistedBuilders():
  """Returns a set of whitelisted builders that the service should process.

  builders are specified in canonical string representations, and following is
  an example config:
  {
    'whitelisted_builders': [
      'chromium/try/linux-rel',
      'chromium/try/linux-chromeos-rel',
    ],
  }
  """
  return set(waterfall_config.GetCodeCoverageSettings().get(
      'whitelisted_builders', []))


def _GetSameOrMostRecentReportForEachPlatform(luci_project, host, project, ref,
                                              revision):
  """Find the matching report on other platforms, or the most recent.

  The intent of this function is to help the UI list the platforms that are
  available, and let the user switch. If a report with the same revision exists
  and is supposed to be visible to the public users, use it, otherwise use the
  most recent visible one.
  """
  result = {}
  for platform, info in _GetPostsubmitPlatformInfoMap(luci_project).iteritems():
    # Some 'platforms' are hidden from the selection to avoid confusion, as they
    # may be custom reports that do not make sense outside a certain team.
    # They should still be reachable via a url.
    if (info.get('hidden') and not users.is_current_user_admin()):
      continue

    bucket = info['bucket']
    builder = info['builder']
    same_report = PostsubmitReport.Get(
        server_host=host,
        project=project,
        ref=ref,
        revision=revision,
        bucket=bucket,
        builder=builder)
    if same_report and same_report.visible:
      result[platform] = same_report
      continue

    query = PostsubmitReport.query(
        PostsubmitReport.gitiles_commit.server_host == host,
        PostsubmitReport.gitiles_commit.project == project,
        PostsubmitReport.bucket == bucket, PostsubmitReport.builder == builder,
        PostsubmitReport.visible == True).order(
            -PostsubmitReport.commit_timestamp)
    entities = query.fetch(limit=1)
    if entities:
      result[platform] = entities[0]

  return result


def _MakePlatformSelect(luci_project, host, project, ref, revision, path,
                        current_platform):
  """Populate values needed to render a form to let the user switch platforms.

  This will produce parameters needed for the form to post to the same page so
  that upon submission it loads the report at the same path, and it will also
  provide the options that can be selected in the dropdown.
  """
  result = {
      'params': {
          'host': host,
          'project': project,
          'ref': ref,
      },
      'options': [],
  }
  if path:
    result['params']['path'] = path
  for platform, report in _GetSameOrMostRecentReportForEachPlatform(
      luci_project, host, project, ref, revision).iteritems():
    value = platform
    if report.gitiles_commit.revision == revision:
      # If the same revision is available in the target platform, append it to
      # the platform name s.t. the form can populate this revision field before
      # submission.
      value = '%s#%s' % (platform, revision)
    result['options'].append({
        'value':
            value,
        'ui_name':
            _GetPostsubmitPlatformInfoMap(luci_project)[platform]['ui_name'],
        'selected':
            platform == current_platform,
    })
  return result


def _GetValidatedData(gs_path):  # pragma: no cover.
  """Returns the json data from the given GS path after validation.

  Args:
    gs_path (str): Path to the file, in the format /bucket/object.

  Returns:
    json_data (dict): the json data of the file pointed by the given GS url, or
        None if the data can't be retrieved.
  """
  logging.info('Fetching data from %s', gs_path)
  content = _GetFileContentFromGs(gs_path)
  assert content, 'Failed to fetch coverage json data from %s' % gs_path

  logging.info('Decompressing and loading coverage data...')
  decompressed_data = zlib.decompress(content)

  del content  # Explicitly release memory.
  data = json.loads(decompressed_data)
  del decompressed_data  # Explicitly release memory.
  logging.info('Finished decompressing and loading coverage data.')

  # Validate that the data is in good format.
  logging.info('Validating coverage data...')
  report = CoverageReport()
  json_format.ParseDict(data, report, ignore_unknown_fields=False)
  del report  # Explicitly delete the proto message to release memory.
  logging.info('Finished validating coverage data.')

  return data


def _RetrieveManifest(repo_url, revision, os_platform):  # pragma: no cover.
  """Returns the manifest of all the dependencies for the given revision.

  Args:
    repo_url (str): The url to the Gitiles project of the root repository.
    revision (str): The revision of the root repository.
    os_platform (str): The platform of the code checkout.

  Returns:
    A list of DependencyRepository instances ordered reversely by the relative
    path of each dependency checkout in the checkout of the root repository.
    The longer the relative path, the smaller index in the returned list.

    The reverse order is to make it easy to reliably determine which dependency
    a file is from, when given a file path relative to the root repository.
  """
  manifest = []

  root_dir = 'src/'

  def AddDependencyToManifest(path, url, revision):  # pragma: no cover.
    if path.startswith(root_dir):
      path = path[len(root_dir):]
    assert not path.startswith('//')
    path = '//' + path
    if not path.endswith('/'):
      path = path + '/'

    # Parse the url to extract the hostname and project name.
    # For "https://chromium.google.com/chromium/src.git", we get
    # ParseResult(netloc='chromium.google.com', path='/chromium/src.git', ...)
    result = urlparse.urlparse(url)
    assert result.path, 'No project extracted from %s' % url

    manifest.append(
        DependencyRepository(
            path=path,
            server_host=result.netloc,
            project=result.path[1:],  # Strip the leading '/'.
            revision=revision))

  # Add the root repository.
  AddDependencyToManifest('src/', repo_url, revision)

  # Add all the dependent repositories.
  # DEPS fetcher now assumes chromium/src and master branch.
  dep_fetcher = chrome_dependency_fetcher.ChromeDependencyFetcher(
      CachedGitilesRepository.Factory(FinditHttpClient()))
  deps = dep_fetcher.GetDependency(revision, os_platform)
  for path, dep in deps.iteritems():
    # Remove clause when crbug.com/929315 gets fixed.
    if path in _GetBlacklistedDeps().get(repo_url, []):
      continue
    AddDependencyToManifest(path, dep.repo_url, dep.revision)

  manifest.sort(key=lambda x: len(x.path), reverse=True)
  return manifest


def _GetMatchedDependencyRepository(report, file_path):  # pragma: no cover.
  """Gets the matched dependency in the manifest of the report.

  Args:
    report (PostsubmitReport): The report that the file is associated with.
    file_path (str): Source absolute path to the file.

  Returns:
    A DependencyRepository if a matched one is found and it is whitelisted,
    otherwise None.
  """
  assert file_path.startswith('//'), 'All file path should start with "//".'

  dependency = None
  for dep in report.manifest:
    if file_path.startswith(dep.path):
      dependency = dep
      break

  if not dependency or dependency.server_host not in _GetAllowedGitilesHosts():
    return None

  return dependency


def _ComposeSourceFileGsPath(report, file_path, revision):
  """Composes a cloud storage path for a specific revision of a source file.

  Args:
    report (PostsubmitReport): The report that the file is associated with.
    file_path (str): Source absolute path to the file.
    revision (str): The gitile revision of the file in its own repo.

  Returns:
    Cloud storage path to the file, in the format /bucket/object. For example,
    /source-files-for-coverage/chromium.googlesource.com/v8/v8/src/date.cc/1234.
  """
  assert file_path.startswith('//'), 'All file path should start with "//".'
  assert revision, 'A valid revision is required'

  dependency = _GetMatchedDependencyRepository(report, file_path)
  assert dependency, (
      '%s file does not belong to any dependency repository' % file_path)

  # Calculate the relative path to the root of the dependency repository itself.
  relative_file_path = file_path[len(dependency.path):]
  return '/%s/%s/%s/%s/%s' % (_SOURCE_FILE_GS_BUCKET, dependency.server_host,
                              dependency.project, relative_file_path, revision)


def _IsFileAvailableInGs(gs_path):  # pragma: no cover.
  """Returns True if the specified object exists, otherwise False.

  Args:
    gs_path (str): Path to the file, in the format /bucket/object.

  Returns:
    True if the object exists, otherwise False.
  """
  try:
    _ = cloudstorage.stat(gs_path)
    return True
  except cloudstorage.NotFoundError:
    return False


def _GetFileContentFromGs(gs_path):  # pragma: no cover.
  """Reads the content of a file in cloud storage.

  This method is more expensive than |_IsFileAvailableInGs|, so if the goal is
  to check if a file exists, |_IsFileAvailableInGs| is preferred.

  Args:
    gs_path (str): Path to the file, in the format /bucket/object.

  Returns:
    The content of the file if it exists, otherwise None."""
  try:
    with cloudstorage.open(gs_path) as f:
      return f.read()
  except cloudstorage.NotFoundError:
    return None


def _WriteFileContentToGs(gs_path, content):  # pragma: no cover.
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


def _GetFileContentFromGitiles(report, file_path,
                               revision):  # pragma: no cover.
  """Fetches the content of a specific revision of a file from gitiles.

  Args:
    report (PostsubmitReport): The report that the file is associated with.
    file_path (str): Source absolute path to the file.
    revision (str): The gitile revision of the file.

  Returns:
    The content of the source file."""
  assert file_path.startswith('//'), 'All file path should start with "//".'
  assert revision, 'A valid revision is required'

  dependency = _GetMatchedDependencyRepository(report, file_path)
  assert dependency, (
      '%s file does not belong to any dependency repository' % file_path)

  # Calculate the relative path to the root of the dependency repository itself.
  relative_file_path = file_path[len(dependency.path):]
  repo = CachedGitilesRepository(FinditHttpClient(), dependency.project_url)
  return repo.GetSource(relative_file_path, revision)


def _IsReportSuspicious(report):
  """Returns True if the newly generated report is suspicious to be incorrect.

  A report is determined to be suspicious if and only if the absolute difference
  between its line coverage percentage and the most recent visible report is
  greater than 1.00%.

  Args:
    report (PostsubmitReport): The report to be evaluated.

  Returns:
    True if the report is suspicious, otherwise False.
  """

  def _GetLineCoveragePercentage(report):  # pragma: no cover
    line_coverage_percentage = None
    summary = report.summary_metrics
    for feature_summary in summary:
      if feature_summary['name'] == 'line':
        line_coverage_percentage = float(
            feature_summary['covered']) / feature_summary['total']
        break

    assert line_coverage_percentage is not None, (
        'Given report has invalid summary')
    return line_coverage_percentage

  target_server_host = report.gitiles_commit.server_host
  target_project = report.gitiles_commit.project
  target_bucket = report.bucket
  target_builder = report.builder
  most_recent_visible_reports = PostsubmitReport.query(
      PostsubmitReport.gitiles_commit.server_host == target_server_host,
      PostsubmitReport.gitiles_commit.project == target_project,
      PostsubmitReport.bucket == target_bucket,
      PostsubmitReport.builder == target_builder, PostsubmitReport.visible ==
      True).order(-PostsubmitReport.commit_timestamp).fetch(1)
  if not most_recent_visible_reports:
    logging.warn('No existing visible reports to use for reference, the new '
                 'report is determined as not suspicious by default')
    return False

  most_recent_visible_report = most_recent_visible_reports[0]
  if abs(
      _GetLineCoveragePercentage(report) -
      _GetLineCoveragePercentage(most_recent_visible_report)) > 0.01:
    return True

  return False


class FetchSourceFile(BaseHandler):
  PERMISSION_LEVEL = Permission.APP_SELF

  def HandlePost(self):
    report_key = self.request.get('report_key')
    path = self.request.get('path')
    revision = self.request.get('revision')

    assert report_key, 'report_key is required'
    assert path, 'path is required'
    assert revision, 'revision is required'

    report = entity_util.GetEntityFromUrlsafeKey(report_key)
    assert report, (
        'Postsubmit report does not exist for urlsafe key' % report_key)

    file_content = _GetFileContentFromGitiles(report, path, revision)
    if not file_content:
      logging.error(
          'Failed to get file from gitiles for %s@%s' % (path, revision))
      return

    gs_path = _ComposeSourceFileGsPath(report, path, revision)
    _WriteFileContentToGs(gs_path, file_content)


class ProcessCodeCoverageData(BaseHandler):
  PERMISSION_LEVEL = Permission.APP_SELF

  def _ProcessFullRepositoryData(self, commit, data, full_gs_metadata_dir,
                                 builder, build_id):

    # Load the commit log first so that we could fail fast before redo all.
    repo_url = 'https://%s/%s.git' % (commit.host, commit.project)
    change_log = CachedGitilesRepository(FinditHttpClient(),
                                         repo_url).GetChangeLog(commit.id)
    assert change_log is not None, 'Failed to retrieve the commit log'

    # Load the manifest based on the DEPS file.
    # TODO(crbug.com/921714): output the manifest as a build output property.
    manifest = _RetrieveManifest(repo_url, commit.id, 'unix')

    report = PostsubmitReport.Create(
        server_host=commit.host,
        project=commit.project,
        ref=commit.ref,
        revision=commit.id,
        bucket=builder.bucket,
        builder=builder.builder,
        commit_timestamp=change_log.committer.time,
        manifest=manifest,
        summary_metrics=data.get('summaries'),
        build_id=build_id,
        visible=False)
    report.put()

    # Save the file-level, directory-level and line-level coverage data.
    for data_type in ('dirs', 'components', 'files', 'file_shards'):
      sub_data = data.get(data_type)
      if not sub_data:
        continue

      logging.info('Processing %d entries for %s', len(sub_data), data_type)

      actual_data_type = data_type
      if data_type == 'file_shards':
        actual_data_type = 'files'

      def FlushEntries(entries, total, last=False):
        # Flush the data in a batch and release memory.
        if len(entries) < 100 and not (last and entries):
          return entries, total

        ndb.put_multi(entries)
        total += len(entries)
        logging.info('Dumped %d coverage data entries of type %s', total,
                     actual_data_type)

        return [], total

      def IterateOverFileShards(file_shards):
        for file_path in file_shards:
          url = '%s/%s' % (full_gs_metadata_dir, file_path)
          # Download data one by one.
          yield _GetValidatedData(url).get('files', [])

      if data_type == 'file_shards':
        data_iterator = IterateOverFileShards(sub_data)
      else:
        data_iterator = [sub_data]

      entities = []
      total = 0

      component_summaries = []
      for dataset in data_iterator:
        for group_data in dataset:
          if actual_data_type == 'components':
            component_summaries.append({
                'name': group_data['path'],
                'path': group_data['path'],
                'summaries': group_data['summaries'],
            })

          if actual_data_type == 'files' and 'revision' in group_data:
            self._FetchAndSaveFileIfNecessary(report, group_data['path'],
                                              group_data['revision'])

          if actual_data_type == 'files':
            coverage_data = FileCoverageData.Create(
                server_host=commit.host,
                project=commit.project,
                ref=commit.ref,
                revision=commit.id,
                path=group_data['path'],
                bucket=builder.bucket,
                builder=builder.builder,
                data=group_data)
          else:
            coverage_data = SummaryCoverageData.Create(
                server_host=commit.host,
                project=commit.project,
                ref=commit.ref,
                revision=commit.id,
                data_type=actual_data_type,
                path=group_data['path'],
                bucket=builder.bucket,
                builder=builder.builder,
                data=group_data)

          entities.append(coverage_data)
          entities, total = FlushEntries(entities, total, last=False)
        del dataset  # Explicitly release memory.
      FlushEntries(entities, total, last=True)

      if component_summaries:
        component_summaries.sort(key=lambda x: x['path'])
        SummaryCoverageData.Create(
            server_host=commit.host,
            project=commit.project,
            ref=commit.ref,
            revision=commit.id,
            data_type='components',
            path='>>',
            bucket=builder.bucket,
            builder=builder.builder,
            data={
                'dirs': component_summaries,
                'path': '>>'
            }).put()
        component_summaries = []
        logging.info('Summary of all components are saved to datastore.')

    if not _IsReportSuspicious(report):
      report.visible = True
      report.put()

      monitoring.code_coverage_full_reports.increment({
          'host':
              commit.host,
          'project':
              commit.project,
          'ref':
              commit.ref or 'refs/heads/master',
          'builder':
              '%s/%s/%s' % (builder.project, builder.bucket, builder.builder),
      })

    monitoring.code_coverage_report_timestamp.set(
        int(time.time()),
        fields={
            'host':
                commit.host,
            'project':
                commit.project,
            'ref':
                commit.ref or 'refs/heads/master',
            'builder':
                '%s/%s/%s' % (builder.project, builder.bucket, builder.builder),
            'is_success':
                report.visible,
        })

  def _FetchAndSaveFileIfNecessary(self, report, path, revision):
    """Fetches the file from gitiles and store to cloud storage if not exist.

    Args:
      report (PostsubmitReport): The report that the file is associated with.
      path (str): Source absolute path to the file.
      revision (str): The gitile revision of the file in its own repo.
    """
    # Due to security concerns, don't cache source files for internal projects.
    if IsInternalInstance():
      return

    assert path.startswith('//'), 'All file path should start with "//"'
    assert revision, 'A valid revision is required'

    gs_path = _ComposeSourceFileGsPath(report, path, revision)
    if _IsFileAvailableInGs(gs_path):
      return

    # Fetch the source files from gitile and save it in gs so that coverage
    # file view can be quickly rendered.
    url = ('/coverage/task/fetch-source-file')
    params = {
        'report_key': report.key.urlsafe(),
        'path': path,
        'revision': revision
    }
    taskqueue.add(
        method='POST',
        url=url,
        target='code-coverage-backend',
        queue_name='code-coverage-fetch-source-file',
        params=params)

  def _ProcessCLPatchData(self, patch, coverage_data):
    """Processes and updates coverage data for per-cl build.

    Part of the responsibility of this method is to calculate per-file coverage
    percentage for the following use cases:
    1. Surface them on Gerrit to provide an overview of the test coverage of
       the CL for authors and reviewers.
    2. For metrics tracking to understand the impact of the coverage data.

    Args:
      patch (buildbucket.v2.GerritChange): A gerrit change with fields: host,
                                           project, change, patchset.
      coverage_data (list): A list of File in coverage proto.
      build_id (int): Id of the build to process coverage data for.
    """

    @ndb.tasklet
    @ndb.transactional
    def _UpdateCoverageDataAsync():
      entity = yield PresubmitCoverageData.GetAsync(
          server_host=patch.host, change=patch.change, patchset=patch.patchset)
      if entity:
        entity.data = code_coverage_util.MergeFilesCoverageDataForPerCL(
            entity.data, coverage_data)
      else:
        entity = PresubmitCoverageData.Create(
            server_host=patch.host,
            change=patch.change,
            patchset=patch.patchset,
            data=coverage_data)
      entity.absolute_percentages = (
          code_coverage_util.CalculateAbsolutePercentages(entity.data))
      entity.incremental_percentages = (
          code_coverage_util.CalculateIncrementalPercentages(
              patch.host, patch.project, patch.change, patch.patchset,
              entity.data))
      yield entity.put_async()

    update_future = _UpdateCoverageDataAsync()

    # Following code invalidates the dependent patchsets whenever the coverage
    # data of the current patchset changes, and it is based on the assumption
    # that the coverage data of the dependent patchsets is always a subset of
    # the current patchset.
    #
    # There is one scenario where the above mentioned assumption doesn't hold:
    # 1. User triggers builder1 on ps1, so ps1 has builder1's coverage data.
    # 2. Ps2 is a trivial-rebase of ps1, and once its coverage data is
    #    requested, it reuses ps1's, which is to say that ps2 now has builder1's
    #    coverage data.
    # 3. User triggers builder2 on ps2, so ps2 contains coverage data from both
    #    builder1 and builder2.
    # 4. User triggers builder3 on ps1, so now ps1 has builder1 and builder3's
    #    coverage data, and it also invalidates ps2, but it's NOT entirely
    #    correct because ps2 has something (builder2) that ps1 doesn't have.
    #
    # In practice, the described scenario is rather extreme corner case because:
    # 1. Most users triggers cq dry run instead of specific builders.
    # 2. When users upload a new trivial-rebase patchset, most likely they'll
    #    never go back to previous patchset to trigger builds.
    #
    # Therefore, it makes sense to do nothing about it for now.
    delete_futures = ndb.delete_multi_async(
        PresubmitCoverageData.query(
            PresubmitCoverageData.cl_patchset.server_host == patch.host,
            PresubmitCoverageData.cl_patchset.change == patch.change,
            PresubmitCoverageData.based_on == patch.patchset).fetch(
                keys_only=True))

    update_future.get_result()
    for f in delete_futures:
      f.get_result()

  def _processCodeCoverageData(self, build_id):
    build = GetV2Build(
        build_id,
        fields=FieldMask(paths=['id', 'output.properties', 'input', 'builder']))

    if not build:
      return BaseHandler.CreateError(
          'Could not retrieve build #%d from buildbucket, retry' % build_id,
          404)

    builder_id = '%s/%s/%s' % (build.builder.project, build.builder.bucket,
                               build.builder.builder)
    if builder_id not in _GetWhitelistedBuilders():
      logging.info('%s is not whitelisted', builder_id)
      return

    # Convert the Struct to standard dict, to use .get, .iteritems etc.
    properties = dict(build.output.properties.items())
    gs_bucket = properties.get('coverage_gs_bucket')
    gs_metadata_dir = properties.get('coverage_metadata_gs_path')
    if properties.get('process_coverage_data_failure'):
      monitoring.code_coverage_cq_errors.increment({
          'project': build.builder.project,
          'bucket': build.builder.bucket,
          'builder': build.builder.builder,
      })

    # Ensure that the coverage data is ready.
    if not gs_bucket or not gs_metadata_dir:
      logging.warn('coverage GS bucket info not available in %r', build.id)
      return

    full_gs_metadata_dir = '/%s/%s' % (gs_bucket, gs_metadata_dir)
    all_json_gs_path = '%s/all.json.gz' % full_gs_metadata_dir
    data = _GetValidatedData(all_json_gs_path)

    if 'coverage_is_presubmit' not in properties:
      logging.error('Expecting "coverage_is_presubmit" in output properties')
      return

    if properties['coverage_is_presubmit']:
      # For presubmit coverage, save the whole data in json.
      # Assume there is only 1 patch which is true in CQ.
      assert len(build.input.gerrit_changes) == 1, 'Expect only one patchset'
      patch = build.input.gerrit_changes[0]
      self._ProcessCLPatchData(patch, data['files'])
    else:
      # For postsubmit coverage, we save the data by file and directory.
      if not self._IsGitilesCommitAvailable(build.input.gitiles_commit):
        self._SetGitilesCommitFromOutputProperty(build, properties)

      assert self._IsGitilesCommitAvailable(build.input.gitiles_commit), (
          'gitiles commit information is expected to be available either in '
          'input properties or output properties')
      self._ProcessFullRepositoryData(build.input.gitiles_commit, data,
                                      full_gs_metadata_dir, build.builder,
                                      build_id)

  def _IsGitilesCommitAvailable(self, gitiles_commit):
    """Returns True if gitiles_commit is available in the input property."""
    return (gitiles_commit.host and gitiles_commit.project and
            gitiles_commit.ref and gitiles_commit.id)

  def _SetGitilesCommitFromOutputProperty(self, build, output_properties):
    """Set gitiles_commit of the build from output properties."""
    logging.info('gitiles_commit is not available in the input properties, '
                 'set them from output properties.')
    build.input.gitiles_commit.host = output_properties.get(
        'gitiles_commit_host')
    build.input.gitiles_commit.project = output_properties.get(
        'gitiles_commit_project')
    build.input.gitiles_commit.ref = output_properties.get('gitiles_commit_ref')
    build.input.gitiles_commit.id = output_properties.get('gitiles_commit_id')

  def HandlePost(self):
    """Loads the data from GS bucket, and dumps them into ndb."""
    logging.info('Processing: %s', self.request.path)
    match = _BUILD_ID_REGEX.match(self.request.path)
    if not match:
      logging.info('Build id not found')
      return

    build_id = int(match.group(1))
    return self._processCodeCoverageData(build_id)

  def HandleGet(self):
    return self.HandlePost()  # For local testing purpose.


def _IsServePresubmitCoverageDataEnabled():
  """Returns True if the feature to serve presubmit coverage data is enabled.

  Returns:
    Returns True if it is enabled, otherwise, False.
  """
  # Unless the flag is explicitly set, assuming disabled by default.
  return waterfall_config.GetCodeCoverageSettings().get(
      'serve_presubmit_coverage_data', False)


def _GetBanner(project):
  """If there is a service banner for a given project landing page, return it.

  E.g. a maintenance announcement or outage acknowledgement, etc.

  The setting is expected to be a dict mapping a project to the contents of the
  div tag for the banner. If no project banner is defined, return the default
  one.

  This expected to be None if no banner is to be shown.
  """
  banners = waterfall_config.GetCodeCoverageSettings().get(
      'project_banners', {})
  return banners.get(project, banners.get('default'))


def _GetPathRootAndSeparatorFromDataType(data_type):
  """Returns the path of the root and path separator for the given data type."""
  if data_type in ('files', 'dirs'):
    return '//', '/'
  elif data_type == 'components':
    return '>>', '>'
  return None, None


def _GetNameToPathSeparator(path, data_type):
  """Returns a list of [name, sub_path] for the given path.

  Example:
  1. //root/src/file.cc  -> [
       ['root/', '//root/'],
       ['src/', '//root/src/'],
       ['file.cc', '//root/src/file.cc']
     ]
  2. //root/src/path1/ -> [
       ['root/', '//root/'],
       ['src/', '//root/src/'],
       ['path1/', '//root/src/path1/']
     ]
  3. component1>component2  -> [
       ['component1', 'component1'],
       ['component2', 'component1>component2'],
     ]
  """
  path_parts = []
  if not path:
    return path_parts

  path_root, path_separator = _GetPathRootAndSeparatorFromDataType(data_type)
  if path == path_root:
    return path_parts

  if data_type == 'components':
    index = 0
  else:
    index = 2  # Skip the leading '//' in the path.

  while index >= 0:
    next_index = path.find(path_separator, index)
    if next_index >= 0:
      name = path[index:next_index + 1]
      if data_type == 'components':
        sub_path = path[:next_index]
      else:
        sub_path = path[:next_index + 1]
      next_index += 1
    else:
      name = path[index:]
      sub_path = path
    path_parts.append([name, sub_path])
    index = next_index

  return path_parts


def _SplitLineIntoRegions(line, uncovered_blocks):
  """Returns a list of regions for a line of code.

  The structure of the output is as follows:
  [
    {
      'covered': True/False # Whether this region is actually covered.
      'text': string # The source text for this region.
    }
  ]

  The regions in the output list are in the order they appear in the line.
  For example, the following loop reconstructs the entire line:

  text = ''
  for region in _SplitLineIntoRegions(line, uncovered_blocks):
    text += region['text']
  assert text == line
  """
  if not uncovered_blocks:
    return [{'is_covered': True, 'text': line}]

  regions = []
  region_start = 0
  for block in uncovered_blocks:
    # Change from 1-indexing to 0-indexing
    first = block['first'] - 1
    last = block['last']
    if last < 0:
      last = len(line)
    else:
      last -= 1

    # Generate the covered region that precedes this uncovered region.
    preceding_text = line[region_start:first]
    if preceding_text:
      regions.append({'is_covered': True, 'text': preceding_text})
    regions.append({
        'is_covered': False,
        # `last` is inclusive
        'text': line[first:last + 1]
    })
    region_start = last + 1

  # If there is any text left on the line, it must be covered. If it were
  # uncovered, it would have been part of the final entry in uncovered_blocks.
  remaining_text = line[region_start:]
  if remaining_text:
    regions.append({'is_covered': True, 'text': remaining_text})

  return regions


class ServeCodeCoverageData(BaseHandler):
  PERMISSION_LEVEL = Permission.ANYONE

  def _ServePerCLCoverageData(self):
    """Serves per-cl coverage data.

    There are two types of requests: 'lines' and 'percentages', and the reason
    why they're separate is that:
    1. Calculating lines takes much longer than percentages, especially when
       data needs to be shared between two equivalent patchsets, while for
       percentages, it's assumed that incremental coverage percentages would be
       the same for equivalent patchsets and no extra work is needed.
    2. Percentages are usually requested much earlier than lines by the Gerrit
       plugin because the later won't be displayed until the user actually
       expands the diff view.

    The format of the returned data conforms to:
    https://chromium.googlesource.com/infra/gerrit-plugins/code-coverage/+/213d226a5f1b78c45c91d49dbe32b09c5609e9bd/src/main/resources/static/coverage.js#93
    """

    def _ServeLines(lines_data):
      """Serves lines coverage data."""
      formatted_data = {'files': []}
      for file_data in lines_data:
        formatted_data['files'].append({
            'path':
                file_data['path'][2:],
            'lines':
                code_coverage_util.DecompressLineRanges(file_data['lines']),
        })

      return {'data': {'data': formatted_data,}, 'allowed_origin': '*'}

    def _ServePercentages(abs_coverage, inc_coverage):
      """Serves percentages coverage data."""
      path_to_abs_coverage = {}
      path_to_inc_coverage = {}
      for e in abs_coverage:
        path_to_abs_coverage[e.path] = {
            'covered': e.covered_lines,
            'total': e.total_lines,
        }
      for e in inc_coverage:
        path_to_inc_coverage[e.path] = {
            'covered': e.covered_lines,
            'total': e.total_lines,
        }

      formatted_data = {'files': []}
      for p in path_to_abs_coverage:
        formatted_data['files'].append({
            'path': p[2:],
            'absolute_coverage': path_to_abs_coverage[p],
            'incremental_coverage': path_to_inc_coverage.get(p, None),
        })

      return {'data': {'data': formatted_data,}, 'allowed_origin': '*'}

    host = self.request.get('host')
    project = self.request.get('project')
    try:
      change = int(self.request.get('change'))
      patchset = int(self.request.get('patchset'))
    except ValueError, ve:
      return BaseHandler.CreateError(
          error_message=(
              'Invalid value for change(%r) or patchset(%r): need int, %s' %
              (self.request.get('change'), self.request.get('patchset'),
               ve.message)),
          return_code=400,
          allowed_origin='*')

    data_type = self.request.get('type', 'lines')

    logging.info('Serving coverage data for CL:')
    logging.info('host=%s', host)
    logging.info('change=%d', change)
    logging.info('patchset=%d', patchset)
    logging.info('type=%s', data_type)

    if host.replace('-review', '') not in _GetAllowedGitilesHosts():
      return BaseHandler.CreateError(
          error_message='Host "%s" is not whitelisted.' % host,
          return_code=400,
          allowed_origin='*')

    if project not in _GetAllowedProjectsForPerCL(host.replace('-review', '')):
      kwargs = {'is_project_supported': False}
      return BaseHandler.CreateError(
          error_message='Project "%s" is not supported.' % project,
          return_code=400,
          allowed_origin='*',
          **kwargs)

    if data_type not in ('lines', 'percentages'):
      return BaseHandler.CreateError(
          error_message=(
              'Invalid type: "%s", must be "lines" (default) or "percentages"' %
              data_type),
          return_code=400,
          allowed_origin='*')

    if not _IsServePresubmitCoverageDataEnabled():
      # TODO(crbug.com/908609): Switch to 'is_service_enabled'.
      kwargs = {'is_project_supported': False}
      return BaseHandler.CreateError(
          error_message='The functionality has been temporarity disabled.',
          return_code=400,
          allowed_origin='*',
          **kwargs)

    entity = PresubmitCoverageData.Get(
        server_host=host, change=change, patchset=patchset)
    is_serving_percentages = (data_type == 'percentages')
    if entity:
      if is_serving_percentages:
        return _ServePercentages(entity.absolute_percentages,
                                 entity.incremental_percentages)

      return _ServeLines(entity.data)

    # If coverage data of the requested patchset is not available, we check
    # previous equivalent patchsets try to reuse their data if applicable.
    equivalent_patchsets = code_coverage_util.GetEquivalentPatchsets(
        host, project, change, patchset)
    if not equivalent_patchsets:
      return BaseHandler.CreateError(
          'Requested coverage data is not found.', 404, allowed_origin='*')

    latest_entity = None
    for ps in sorted(equivalent_patchsets, reverse=True):
      latest_entity = PresubmitCoverageData.Get(
          server_host=host, change=change, patchset=ps)
      if latest_entity and latest_entity.based_on is None:
        break

    if latest_entity is None:
      return BaseHandler.CreateError(
          'Requested coverage data is not found.', 404, allowed_origin='*')

    if is_serving_percentages:
      return _ServePercentages(latest_entity.absolute_percentages,
                               latest_entity.incremental_percentages)

    try:
      rebased_coverage_data = (
          code_coverage_util.RebasePresubmitCoverageDataBetweenPatchsets(
              host=host,
              project=project,
              change=change,
              patchset_src=latest_entity.cl_patchset.patchset,
              patchset_dest=patchset,
              coverage_data_src=latest_entity.data))
    except code_coverage_util.MissingChangeDataException as mcde:
      return BaseHandler.CreateError(
          'Requested coverage data is not found. %s' % mcde.message,
          404,
          allowed_origin='*')

    entity = PresubmitCoverageData.Create(
        server_host=host,
        change=change,
        patchset=patchset,
        data=rebased_coverage_data)
    entity.absolute_percentages = latest_entity.absolute_percentages
    entity.incremental_percentages = latest_entity.incremental_percentages
    entity.based_on = latest_entity.cl_patchset.patchset
    entity.put()
    return _ServeLines(entity.data)

  def _ServeProjectViewCoverageData(self, luci_project, host, project, ref,
                                    revision, platform, bucket, builder):
    """Serves coverage data for the project view."""
    cursor = self.request.get('cursor', None)
    page_size = int(self.request.get('page_size', 100))
    direction = self.request.get('direction', 'next').lower()

    query = PostsubmitReport.query(
        PostsubmitReport.gitiles_commit.server_host == host,
        PostsubmitReport.gitiles_commit.project == project,
        PostsubmitReport.bucket == bucket, PostsubmitReport.builder == builder)
    order_props = [(PostsubmitReport.commit_timestamp, 'desc')]
    entities, prev_cursor, next_cursor = GetPagedResults(
        query, order_props, cursor, direction, page_size)

    # TODO(crbug.com/926237): Move the conversion to client side and use
    # local timezone.
    data = []
    for entity in entities:
      data.append({
          'gitiles_commit': entity.gitiles_commit.to_dict(),
          'commit_timestamp': ConvertUTCToPST(entity.commit_timestamp),
          'summary_metrics': entity.summary_metrics,
          'build_id': entity.build_id,
          'visible': entity.visible,
      })

    return {
        'data': {
            'luci_project':
                luci_project,
            'gitiles_commit': {
                'host': host,
                'project': project,
                'ref': ref,
                'revision': revision,
            },
            'platform':
                platform,
            'platform_ui_name':
                _GetPostsubmitPlatformInfoMap(luci_project)[platform]
                ['ui_name'],
            'metrics':
                code_coverage_util.GetMetricsBasedOnCoverageTool(
                    _GetPostsubmitPlatformInfoMap(luci_project)[platform]
                    ['coverage_tool']),
            'data':
                data,
            'data_type':
                'project',
            'platform_select':
                _MakePlatformSelect(luci_project, host, project, ref, revision,
                                    None, platform),
            'banner':
                _GetBanner(project),
            'next_cursor':
                next_cursor,
            'prev_cursor':
                prev_cursor,
        },
        'template': 'coverage/project_view.html',
    }

  def HandleGet(self):
    if self.request.path == '/coverage/api/coverage-data':
      return self._ServePerCLCoverageData()

    match = _LUCI_PROJECT_REGEX.match(self.request.path)
    if not match:
      return BaseHandler.CreateError('Invalid url path %s' % self.request.path,
                                     400)
    luci_project = match.group(1)

    default_config = _GetPostsubmitDefaultReportConfig(luci_project)
    if not default_config:
      return BaseHandler.CreateError(
          'Default report config is missing for project: "%s", please file a '
          'bug with component: Infra>Test>CodeCoverage for fixing it' %
          luci_project, 400)

    host = self.request.get('host', default_config['host'])
    project = self.request.get('project', default_config['project'])
    ref = self.request.get('ref', default_config['ref'])

    revision = self.request.get('revision')
    path = self.request.get('path')
    data_type = self.request.get('data_type')
    platform = self.request.get('platform', default_config['platform'])
    list_reports = self.request.get('list_reports', False)
    if isinstance(list_reports, str):
      list_reports = (list_reports.lower() == 'true')

    if not data_type and path:
      if path.endswith('/'):
        data_type = 'dirs'
      elif path and '>' in path:
        data_type = 'components'
      else:
        data_type = 'files'

    logging.info('host=%s', host)
    logging.info('project=%s', project)
    logging.info('ref=%s', ref)
    logging.info('revision=%s', revision)
    logging.info('data_type=%s', data_type)
    logging.info('path=%s', path)
    logging.info('platform=%s', platform)

    if not project:
      return BaseHandler.CreateError('Invalid request', 400)

    logging.info('Servicing coverage data for postsubmit')
    platform_info_map = _GetPostsubmitPlatformInfoMap(luci_project)
    if platform not in platform_info_map:
      return BaseHandler.CreateError('Platform: %s is not supported' % platform,
                                     404)
    bucket = platform_info_map[platform]['bucket']
    builder = platform_info_map[platform]['builder']
    warning = platform_info_map[platform].get('warning')

    if list_reports:
      return self._ServeProjectViewCoverageData(
          luci_project, host, project, ref, revision, platform, bucket, builder)

    template = None
    if not data_type:
      data_type = 'dirs'
    if not revision:
      query = PostsubmitReport.query(
          PostsubmitReport.gitiles_commit.server_host == host,
          PostsubmitReport.gitiles_commit.project == project,
          PostsubmitReport.bucket == bucket,
          PostsubmitReport.builder == builder, PostsubmitReport.visible ==
          True).order(-PostsubmitReport.commit_timestamp)
      entities = query.fetch(limit=1)
      report = entities[0]
      revision = report.gitiles_commit.revision

    else:
      report = PostsubmitReport.Get(
          server_host=host,
          project=project,
          ref=ref,
          revision=revision,
          bucket=bucket,
          builder=builder)
      if not report:
        return BaseHandler.CreateError('Report record not found', 404)

    template = 'coverage/summary_view.html'
    if data_type == 'dirs':
      default_path = '//'
    elif data_type == 'components':
      default_path = '>>'
    else:
      if data_type != 'files':
        return BaseHandler.CreateError(
            'Expected data_type to be "files", but got "%s"' % data_type, 400)

      template = 'coverage/file_view.html'

    path = path or default_path

    if data_type == 'files':
      entity = FileCoverageData.Get(
          server_host=host,
          project=project,
          ref=ref,
          revision=revision,
          path=path,
          bucket=bucket,
          builder=builder)
      if not entity:
        warning = ('File "%s" does not exist in this report, defaulting to root'
                   % path)
        logging.warning(warning)
        path = '//'
        data_type = 'dirs'
        template = 'coverage/summary_view.html'
    if data_type != 'files':
      entity = SummaryCoverageData.Get(
          server_host=host,
          project=project,
          ref=ref,
          revision=revision,
          data_type=data_type,
          path=path,
          bucket=bucket,
          builder=builder)
      if not entity:
        warning = ('Path "%s" does not exist in this report, defaulting to root'
                   % path)
        logging.warning(warning)
        path = default_path
        entity = SummaryCoverageData.Get(
            server_host=host,
            project=project,
            ref=ref,
            revision=revision,
            data_type=data_type,
            path=path,
            bucket=bucket,
            builder=builder)

    metadata = entity.data
    data = {
        'metadata': metadata,
    }

    line_to_data = None
    if data_type == 'files':
      line_to_data = collections.defaultdict(dict)

      if 'revision' in metadata:
        gs_path = _ComposeSourceFileGsPath(report, path, metadata['revision'])
        file_content = _GetFileContentFromGs(gs_path)
        if not file_content:
          # Fetching files from Gitiles is slow, only use it as a backup.
          file_content = _GetFileContentFromGitiles(report, path,
                                                    metadata['revision'])
      else:
        # If metadata['revision'] is empty, it means that the file is not
        # a source file.
        file_content = None

      if not file_content:
        line_to_data[1]['line'] = '!!!!No source code available!!!!'
        line_to_data[1]['count'] = 0
      else:
        file_lines = file_content.splitlines()
        for i, line in enumerate(file_lines):
          # According to http://jinja.pocoo.org/docs/2.10/api/#unicode,
          # Jinja requires passing unicode objects or ASCII-only bytestring,
          # and given that it is possible for source files to have non-ASCII
          # chars, thus converting lines to unicode.
          line_to_data[i + 1]['line'] = unicode(line, 'utf8')
          line_to_data[i + 1]['count'] = -1

        uncovered_blocks = {}
        if 'uncovered_blocks' in metadata:
          for line_data in metadata['uncovered_blocks']:
            uncovered_blocks[line_data['line']] = line_data['ranges']

        for line in metadata['lines']:
          for line_num in range(line['first'], line['last'] + 1):
            line_to_data[line_num]['count'] = line['count']
            if line_num in uncovered_blocks:
              text = line_to_data[line_num]['line']
              regions = _SplitLineIntoRegions(text, uncovered_blocks[line_num])
              line_to_data[line_num]['regions'] = regions
              line_to_data[line_num]['is_partially_covered'] = True
            else:
              line_to_data[line_num]['is_partially_covered'] = False

      line_to_data = list(line_to_data.iteritems())
      line_to_data.sort(key=lambda x: x[0])
      data['line_to_data'] = line_to_data

    # Compute the mapping of the name->path mappings in order.
    path_parts = _GetNameToPathSeparator(path, data_type)
    path_root, _ = _GetPathRootAndSeparatorFromDataType(data_type)
    return {
        'data': {
            'luci_project':
                luci_project,
            'gitiles_commit': {
                'host': host,
                'project': project,
                'ref': ref,
                'revision': revision,
            },
            'path':
                path,
            'platform':
                platform,
            'platform_ui_name':
                _GetPostsubmitPlatformInfoMap(luci_project)[platform]
                ['ui_name'],
            'path_root':
                path_root,
            'metrics':
                code_coverage_util.GetMetricsBasedOnCoverageTool(
                    _GetPostsubmitPlatformInfoMap(luci_project)[platform]
                    ['coverage_tool']),
            'data':
                data,
            'data_type':
                data_type,
            'path_parts':
                path_parts,
            'platform_select':
                _MakePlatformSelect(luci_project, host, project, ref, revision,
                                    path, platform),
            'banner':
                _GetBanner(project),
            'warning':
                warning,
        },
        'template': template,
    }
