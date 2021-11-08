# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.from datetime import datetime

import json
import logging
import re
import time
import urlparse
import zlib

import cloudstorage
from google.appengine.api import taskqueue
from google.appengine.ext import ndb
from google.protobuf import json_format
from google.protobuf.field_mask_pb2 import FieldMask

from common import constants
from common import monitoring
from common.findit_http_client import FinditHttpClient
from common.waterfall.buildbucket_client import GetV2Build
from gae_libs.appengine_util import IsInternalInstance
from gae_libs.handlers.base_handler import BaseHandler, Permission
from gae_libs.gitiles.cached_gitiles_repository import CachedGitilesRepository
from handlers.code_coverage import utils
from libs.deps import chrome_dependency_fetcher
from model.code_coverage import DependencyRepository
from model.code_coverage import FileCoverageData
from model.code_coverage import PostsubmitReport
from model.code_coverage import PresubmitCoverageData
from model.code_coverage import SummaryCoverageData
from model.proto.gen.code_coverage_pb2 import CoverageReport
from services.code_coverage import code_coverage_util
from waterfall import waterfall_config

# The regex to extract the build id from the url path.
_BUILD_ID_REGEX = re.compile(r'.*/build/(\d+)$')


def _AddDependencyToManifest(path, url, revision,
                             manifest):  # pragma: no cover.
  """Adds a dependency to the given manifest.

  Args:
    path (str): Path to the dependency repo.
    url (str): The url to the Gitiles project of the root repository.
    revision (str): The revision of the root repository.
    manifest: A list of DependencyRepository.
  """
  assert path.startswith('//')
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


def _GetDisallowedDeps():
  """Returns a map of disallowed dependencies to skip adding to manifest.

  Main use case is to skip dependency repos that have malformed structures, and
  the mapping is from root repo url to list of dependency paths (relative to
  the root of the checkout).
  """
  return waterfall_config.GetCodeCoverageSettings().get('blacklisted_deps', {})


def _RetrieveChromeManifest(repo_url, revision,
                            os_platform):  # pragma: no cover.
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

  # Add the root repository.
  _AddDependencyToManifest('//', repo_url, revision, manifest)

  # Add all the dependent repositories.
  # DEPS fetcher now assumes chromium/src and main branch.
  dep_fetcher = chrome_dependency_fetcher.ChromeDependencyFetcher(
      CachedGitilesRepository.Factory(FinditHttpClient()))
  deps = dep_fetcher.GetDependency(revision, os_platform)
  for path, dep in deps.iteritems():
    # Remove clause when crbug.com/929315 gets fixed.
    if path in _GetDisallowedDeps().get(repo_url, []):
      continue

    # Public DEPS paths have the src/ prefix, and they need to be striped to be
    # converted to source absolute path format.
    path = '//' + path[len('src/'):]
    _AddDependencyToManifest(path, dep.repo_url, dep.revision, manifest)

  manifest.sort(key=lambda x: len(x.path), reverse=True)
  return manifest


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


def _GetValidatedData(gs_path):  # pragma: no cover.
  """Returns the json data from the given GS path after validation.

  Args:
    gs_path (str): Path to the file, in the format /bucket/object.

  Returns:
    json_data (dict): the json data of the file pointed by the given GS url, or
        None if the data can't be retrieved.
  """
  logging.info('Fetching data from %s', gs_path)
  content = utils.GetFileContentFromGs(gs_path)
  assert content, 'Failed to fetch coverage json data from %s' % gs_path

  logging.info('Decompressing and loading coverage data...')
  decompressed_data = zlib.decompress(content)

  del content  # Explicitly release memory.
  data = json.loads(decompressed_data)
  del decompressed_data  # Explicitly release memory.
  logging.info('Finished decompressing and loading coverage data.')

  # According to https://developers.google.com/discovery/v1/type-format, certain
  # serialization APIs will automatically convert int64 to string when
  # serializing to JSON, and to facilitate later computations, the following for
  # loops convert them back to int64 (int in Python).
  # The following workaround should be removed when the service migrates away
  # from JSON.
  for file_data in data.get('files', []):
    for line_data in file_data.get('lines', []):
      line_data['count'] = int(line_data['count'])

  # Validate that the data is in good format.
  logging.info('Validating coverage data...')
  report = CoverageReport()
  json_format.ParseDict(data, report, ignore_unknown_fields=False)
  del report  # Explicitly delete the proto message to release memory.
  logging.info('Finished validating coverage data.')

  return data


def _GetAllowedBuilders():
  """Returns a set of allowed builders that the service should process.

  builders are specified in canonical string representations, and following is
  an example config:
  {
    'allowed_builders': [
      'chromium/try/linux-rel',
      'chromium/try/linux-chromeos-rel',
    ]
  }
  """
  return set(waterfall_config.GetCodeCoverageSettings().get(
      'allowed_builders', []))


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
      PostsubmitReport.gitiles_commit.project == target_project,
      PostsubmitReport.gitiles_commit.server_host == target_server_host,
      PostsubmitReport.bucket == target_bucket,
      PostsubmitReport.builder == target_builder,
      PostsubmitReport.visible == True, PostsubmitReport.modifier_id ==
      0).order(-PostsubmitReport.commit_timestamp).fetch(1)
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


class ProcessCodeCoverageData(BaseHandler):
  PERMISSION_LEVEL = Permission.APP_SELF

  def _ProcessFullRepositoryData(self, commit, data, full_gs_metadata_dir,
                                 builder, build_id, mimic_builder_name):
    # Load the commit log first so that we could fail fast before redo all.
    repo_url = 'https://%s/%s.git' % (commit.host, commit.project)
    change_log = CachedGitilesRepository(FinditHttpClient(),
                                         repo_url).GetChangeLog(commit.id)
    assert change_log is not None, 'Failed to retrieve the commit log'

    # TODO(crbug.com/921714): output the manifest as a build output property,
    # and make it project agnostic.
    if (commit.host == 'chromium.googlesource.com' and
        commit.project == 'chromium/src'):
      manifest = _RetrieveChromeManifest(repo_url, commit.id, 'unix')
    else:
      # For projects other than chromium/src, dependency repos are ignored for
      # simplicity.
      manifest = []
      _AddDependencyToManifest('//', repo_url, commit.id, manifest)

    report = PostsubmitReport.Create(
        server_host=commit.host,
        project=commit.project,
        ref=commit.ref,
        revision=commit.id,
        bucket=builder.bucket,
        builder=mimic_builder_name,
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

          if actual_data_type == 'files' and group_data.get('revision', ''):
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
                builder=mimic_builder_name,
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
                builder=mimic_builder_name,
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
            builder=mimic_builder_name,
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
              commit.ref or 'refs/heads/main',
          'builder':
              '%s/%s/%s' %
              (builder.project, builder.bucket, mimic_builder_name),
      })

    monitoring.code_coverage_report_timestamp.set(
        int(time.time()),
        fields={
            'host':
                commit.host,
            'project':
                commit.project,
            'ref':
                commit.ref or 'refs/heads/main',
            'builder':
                '%s/%s/%s' %
                (builder.project, builder.bucket, mimic_builder_name),
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

    gs_path = utils.ComposeSourceFileGsPath(report.manifest, path, revision)
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

  def _ProcessCLPatchData(self, mimic_builder, patch, coverage_data):
    """Processes and updates coverage data for per-cl build.

    Part of the responsibility of this method is to calculate per-file coverage
    percentage for the following use cases:
    1. Surface them on Gerrit to provide an overview of the test coverage of
       the CL for authors and reviewers.
    2. For metrics tracking to understand the impact of the coverage data.

    Args:
      mimic_builder (string): Name of the builder that we are mimicking coverage
                              data belongs to. For example, if linux-rel is
                              producing unit tests coverage, mimic_builder name
                              would be 'linux-rel_unit'.
      patch (buildbucket.v2.GerritChange): A gerrit change with fields: host,
                                           project, change, patchset.
      coverage_data (list): A list of File in coverage proto.
    """

    @ndb.tasklet
    @ndb.transactional
    def _UpdateCoverageDataAsync():

      def _GetEntity(entity):
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
        return entity

      def _GetEntityForUnit(entity):
        if entity:
          entity.data_unit = code_coverage_util.MergeFilesCoverageDataForPerCL(
              entity.data_unit, coverage_data)
        else:
          entity = PresubmitCoverageData.Create(
              server_host=patch.host,
              change=patch.change,
              patchset=patch.patchset,
              data_unit=coverage_data)
        entity.absolute_percentages_unit = (
            code_coverage_util.CalculateAbsolutePercentages(entity.data_unit))
        entity.incremental_percentages_unit = (
            code_coverage_util.CalculateIncrementalPercentages(
                patch.host, patch.project, patch.change, patch.patchset,
                entity.data_unit))
        return entity

      entity = yield PresubmitCoverageData.GetAsync(
          server_host=patch.host, change=patch.change, patchset=patch.patchset)
      # Update/Create entity with unit test coverage fields populated
      # if mimic_builder represents a unit tests only builder.
      if mimic_builder.endswith('_unit'):
        entity = _GetEntityForUnit(entity)
      else:
        entity = _GetEntity(entity)
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

  def _ProcessCodeCoverageData(self, build_id):
    build = GetV2Build(
        build_id,
        fields=FieldMask(paths=['id', 'output.properties', 'input', 'builder']))

    if not build:
      return BaseHandler.CreateError(
          'Could not retrieve build #%d from buildbucket, retry' % build_id,
          404)

    builder_id = '%s/%s/%s' % (build.builder.project, build.builder.bucket,
                               build.builder.builder)
    if builder_id not in _GetAllowedBuilders():
      logging.info('%s is not allowed', builder_id)
      return

    # Convert the Struct to standard dict, to use .get, .iteritems etc.
    properties = dict(build.output.properties.items())
    gs_bucket = properties.get('coverage_gs_bucket')

    gs_metadata_dirs = properties.get('coverage_metadata_gs_paths')

    if properties.get('process_coverage_data_failure'):
      monitoring.code_coverage_cq_errors.increment({
          'project': build.builder.project,
          'bucket': build.builder.bucket,
          'builder': build.builder.builder,
      })

    # Ensure that the coverage data is ready.
    if not gs_bucket or not gs_metadata_dirs:
      logging.warn('coverage GS bucket info not available in %r', build.id)
      return

    if 'coverage_is_presubmit' not in properties:
      logging.error('Expecting "coverage_is_presubmit" in output properties')
      return

    # Get mimic builder names from builder output properties. Multiple test
    # types' coverage data will be uploaded to separated folders, mimicking
    # these come from different builders.
    mimic_builder_names = properties.get('mimic_builder_names')
    if not mimic_builder_names:
      logging.error('Couldn\'t find valid mimic_builder_names property from '
                    'builder output properties.')
      return

    assert (len(mimic_builder_names) == len(gs_metadata_dirs)
           ), 'mimic builder names and gs paths should be of the same length'

    if properties['coverage_is_presubmit']:
      # For presubmit coverage, save the whole data in json.
      # Assume there is only 1 patch which is true in CQ.
      assert len(build.input.gerrit_changes) == 1, 'Expect only one patchset'
      for gs_metadata_dir, mimic_builder_name in zip(gs_metadata_dirs,
                                                     mimic_builder_names):
        full_gs_metadata_dir = '/%s/%s' % (gs_bucket, gs_metadata_dir)
        all_json_gs_path = '%s/all.json.gz' % full_gs_metadata_dir
        data = _GetValidatedData(all_json_gs_path)
        patch = build.input.gerrit_changes[0]
        self._ProcessCLPatchData(mimic_builder_name, patch, data['files'])
    else:
      if (properties.get('coverage_override_gitiles_commit', False) or
          not self._IsGitilesCommitAvailable(build.input.gitiles_commit)):
        self._SetGitilesCommitFromOutputProperty(build, properties)

      assert self._IsGitilesCommitAvailable(build.input.gitiles_commit), (
          'gitiles commit information is expected to be available either in '
          'input properties or output properties')

      for gs_metadata_dir, mimic_builder_name in zip(gs_metadata_dirs,
                                                     mimic_builder_names):
        full_gs_metadata_dir = '/%s/%s' % (gs_bucket, gs_metadata_dir)
        all_json_gs_path = '%s/all.json.gz' % full_gs_metadata_dir
        data = _GetValidatedData(all_json_gs_path)
        self._ProcessFullRepositoryData(build.input.gitiles_commit, data,
                                        full_gs_metadata_dir, build.builder,
                                        build_id, mimic_builder_name)

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
    return self._ProcessCodeCoverageData(build_id)

  def HandleGet(self):
    return self.HandlePost()  # For local testing purpose.
