# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import collections
import logging
import re

from google.appengine.api import users

from gae_libs.handlers.base_handler import BaseHandler, Permission
from gae_libs.dashboard_util import GetPagedResults
from handlers.code_coverage import utils
from libs.time_util import ConvertUTCToPST
from model.code_coverage import CoverageReportModifier
from model.code_coverage import FileCoverageData
from model.code_coverage import PostsubmitReport
from model.code_coverage import PresubmitCoverageData
from model.code_coverage import SummaryCoverageData
from services.code_coverage import code_coverage_util
from waterfall import waterfall_config

# The regex to extract the luci project name from the url path.
_LUCI_PROJECT_REGEX = re.compile(r'^/coverage/p/([^/]+)')

# The regex to extract the year since which referenced coverage is desired
_REFERENCED_COVERAGE_YEAR_REGEX = re.compile(r'.*/referenced([0-9]+)')


def _GetPostsubmitDefaultReportConfig(luci_project):
  """Returns a tuple of (host, project, ref, platform) to serve default report.

  Following is an example config:
  {
    'default_postsubmit_report_config': {
      'chromium': {
        'host': 'chromium.googlesource.com',
        'project': 'chromium/src',
        'ref': 'refs/heads/main',
        'platform': 'linux',
      }
    }
  }
  """
  return waterfall_config.GetCodeCoverageSettings().get(
      'default_postsubmit_report_config', {}).get(luci_project, None)


def _GetSameOrMostRecentReportForEachPlatform(luci_project, host, project, ref,
                                              revision):
  """Find the matching report on other platforms, or the most recent.

  The intent of this function is to help the UI list the platforms that are
  available, and let the user switch. If a report with the same revision exists
  and is supposed to be visible to the public users, use it, otherwise use the
  most recent visible one.
  """
  result = {}
  for platform, info in utils.GetPostsubmitPlatformInfoMap(
      luci_project).iteritems():
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
        PostsubmitReport.gitiles_commit.project == project,
        PostsubmitReport.gitiles_commit.server_host == host,
        PostsubmitReport.bucket == bucket, PostsubmitReport.builder == builder,
        PostsubmitReport.visible == True, PostsubmitReport.modifier_id ==
        0).order(-PostsubmitReport.commit_timestamp)
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
    option = {
        'platform':
            platform,
        'ui_name':
            utils.GetPostsubmitPlatformInfoMap(luci_project)[platform]
            ['ui_name'],
        'selected':
            platform == current_platform,
    }
    if report.gitiles_commit.revision == revision:
      # If the same revision is available in the target platform, add it to the
      # option s.t. the form can populate this revision field before
      # submission.
      option['revision'] = revision
    result['options'].append(option)
  result['options'].sort(key=lambda x: x['ui_name'])
  return result


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
      lines_data = lines_data or []
      formatted_data = {'files': []}
      for file_data in lines_data:
        formatted_data['files'].append({
            'path':
                file_data['path'][2:],
            'lines':
                code_coverage_util.DecompressLineRanges(file_data['lines']),
        })

      return {'data': {'data': formatted_data,}, 'allowed_origin': '*'}

    def _ServePercentages(abs_coverage, inc_coverage, abs_unit_tests_coverage,
                          inc_unit_tests_coverage):
      """Serves percentages coverage data."""

      def _GetCoverageMetricsPerFile(coverage):
        coverage_per_file = {}
        for e in coverage:
          coverage_per_file[e.path] = {
              'covered': e.covered_lines,
              'total': e.total_lines,
          }
        return coverage_per_file

      abs_coverage_per_file = _GetCoverageMetricsPerFile(abs_coverage)
      inc_coverage_per_file = _GetCoverageMetricsPerFile(inc_coverage)
      abs_unit_tests_coverage_per_file = _GetCoverageMetricsPerFile(
          abs_unit_tests_coverage)
      inc_unit_tests_coverage_per_file = _GetCoverageMetricsPerFile(
          inc_unit_tests_coverage)

      formatted_data = {'files': []}
      for p in set(abs_coverage_per_file.keys() +
                   abs_unit_tests_coverage_per_file.keys()):
        formatted_data['files'].append({
            'path':
                p[2:],
            'absolute_coverage':
                abs_coverage_per_file.get(p, None),
            'incremental_coverage':
                inc_coverage_per_file.get(p, None),
            'absolute_unit_tests_coverage':
                abs_unit_tests_coverage_per_file.get(p, None),
            'incremental_unit_tests_coverage':
                inc_unit_tests_coverage_per_file.get(p, None),
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

    configs = utils.GetAllowedGitilesConfigs()
    if project not in configs.get(host.replace('-review', ''), {}):
      return BaseHandler.CreateError(
          error_message='"%s/%s" is not supported.' % (host, project),
          return_code=400,
          allowed_origin='*',
          is_project_supported=False)

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
                                 entity.incremental_percentages,
                                 entity.absolute_percentages_unit,
                                 entity.incremental_percentages_unit)

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
                               latest_entity.incremental_percentages,
                               latest_entity.absolute_percentages_unit,
                               latest_entity.incremental_percentages_unit)

    try:
      rebased_coverage_data = \
        code_coverage_util.RebasePresubmitCoverageDataBetweenPatchsets(
          host=host,
          project=project,
          change=change,
          patchset_src=latest_entity.cl_patchset.patchset,
          patchset_dest=patchset,
          coverage_data_src=latest_entity.data) if latest_entity.data else None
      rebased_coverage_data_unit = \
        code_coverage_util.RebasePresubmitCoverageDataBetweenPatchsets(
          host=host,
          project=project,
          change=change,
          patchset_src=latest_entity.cl_patchset.patchset,
          patchset_dest=patchset,
          coverage_data_src=latest_entity.data_unit
      ) if latest_entity.data_unit else None
    except code_coverage_util.MissingChangeDataException as mcde:
      return BaseHandler.CreateError(
          'Requested coverage data is not found. %s' % mcde.message,
          404,
          allowed_origin='*')

    entity = PresubmitCoverageData.Create(
        server_host=host,
        change=change,
        patchset=patchset,
        data=rebased_coverage_data,
        data_unit=rebased_coverage_data_unit)
    entity.absolute_percentages = latest_entity.absolute_percentages
    entity.incremental_percentages = latest_entity.incremental_percentages
    entity.absolute_percentages_unit = latest_entity.absolute_percentages_unit
    entity.incremental_percentages_unit = \
      latest_entity.incremental_percentages_unit
    entity.based_on = latest_entity.cl_patchset.patchset
    entity.put()
    return _ServeLines(entity.data)

  def _ServeProjectViewCoverageData(self, luci_project, host, project, ref,
                                    revision, platform, bucket, builder,
                                    test_suite_type, modifier_id):
    """Serves coverage data for the project view."""
    cursor = self.request.get('cursor', None)
    page_size = int(self.request.get('page_size', 100))
    direction = self.request.get('direction', 'next').lower()

    query = PostsubmitReport.query(
        PostsubmitReport.gitiles_commit.project == project,
        PostsubmitReport.gitiles_commit.server_host == host,
        PostsubmitReport.bucket == bucket, PostsubmitReport.builder == builder,
        PostsubmitReport.modifier_id == modifier_id)
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

    current_user = users.get_current_user()
    show_invisible_report = (
        current_user.email().endswith('@google.com') if current_user else False)
    metrics = code_coverage_util.GetMetricsBasedOnCoverageTool(
        utils.GetPostsubmitPlatformInfoMap(luci_project)[platform]
        ['coverage_tool'])
    if modifier_id != 0:
      # Only line coverage metric is supported for cases other than
      # default post submit report
      metrics = [x for x in metrics if x['name'] == 'line']
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
                utils.GetPostsubmitPlatformInfoMap(luci_project)[platform]
                ['ui_name'],
            'metrics':
                metrics,
            'data':
                data,
            'data_type':
                'project',
            'test_suite_type':
                test_suite_type,
            'modifier_id':
                modifier_id,
            'platform_select':
                _MakePlatformSelect(luci_project, host, project, ref, revision,
                                    None, platform),
            'banner':
                _GetBanner(project),
            'show_invisible_report':
                show_invisible_report,
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

    def _GetLuciProject(path):
      match = _LUCI_PROJECT_REGEX.match(path)
      return match.group(1) if match else None

    luci_project = _GetLuciProject(self.request.path)
    if not luci_project:
      return BaseHandler.CreateError('Invalid url path %s' % self.request.path,
                                     400)
    logging.info('luci_project=%s', luci_project)
    default_config = _GetPostsubmitDefaultReportConfig(luci_project)
    if not default_config:
      return BaseHandler.CreateError(
          'Default report config is missing for project: "%s", please file a '
          'bug with component: Infra>Test>CodeCoverage for fixing it' %
          luci_project, 400)

    host = self.request.get('host', default_config['host'])
    project = self.request.get('project', default_config['project'])

    def _GetReferencedCoverageYear():
      match = _REFERENCED_COVERAGE_YEAR_REGEX.match(self.request.path)
      return int(match.group(1)) if match else None

    # If the request is for referenced coverage, find the corresponding
    # CoverageReportModifier and redirect with modifier_id in params
    referenced_coverage_year = _GetReferencedCoverageYear()
    if referenced_coverage_year:
      modifier_id = utils.GetLastActiveReferenceCommitForYear(
          host, project, referenced_coverage_year - 1)
      if not modifier_id:
        return BaseHandler.CreateError(
            'No reference commit found for host %s, project %s and year %d' %
            host, project, referenced_coverage_year)
      path = self.request.path[:-len('/referencedYYYY')]
      query_string = self.request.query_string
      if query_string:
        query_string = '%s&modifier_id=%d' % (query_string, modifier_id)
      else:
        query_string = 'modifier_id=%d' % (modifier_id)
      new_url = 'https://%s%s?%s' % (self.request.host, path, query_string)
      return self.CreateRedirect(new_url)

    ref = self.request.get('ref', default_config['ref'])
    revision = self.request.get('revision')
    platform = self.request.get('platform', default_config['platform'])
    list_reports = self.request.get('list_reports', 'False').lower() == 'true'
    path = self.request.get('path')
    test_suite_type = self.request.get('test_suite_type', 'any')
    modifier_id = int(self.request.get('modifier_id', '0'))

    logging.info('host=%s', host)
    logging.info('project=%s', project)
    logging.info('ref=%s', ref)
    logging.info('revision=%s', revision)
    logging.info('path=%s', path)
    logging.info('platform=%s', platform)
    logging.info('test_suite_type=%s' % test_suite_type)
    logging.info('modifier_id=%d' % modifier_id)

    configs = utils.GetAllowedGitilesConfigs()
    if ref not in configs.get(host, {}).get(project, []):
      return BaseHandler.CreateError(
          '"%s/%s/+/%s" is not supported.' % (host, project, ref), 400)

    logging.info('Servicing coverage data for postsubmit')
    platform_info_map = utils.GetPostsubmitPlatformInfoMap(luci_project)
    if platform not in platform_info_map:
      return BaseHandler.CreateError('Platform: %s is not supported' % platform,
                                     400)
    bucket = platform_info_map[platform]['bucket']
    builder = platform_info_map[platform]['builder']
    if test_suite_type == 'unit':
      builder += '_unit'
    warning = platform_info_map[platform].get('warning')

    if list_reports:
      return self._ServeProjectViewCoverageData(luci_project, host, project,
                                                ref, revision, platform, bucket,
                                                builder, test_suite_type,
                                                modifier_id)

    # Get manifest and other key report attributes from the full codebase
    # report at the specified revision. If the revision is not specified,
    # get the required info from the latest full code base coverage reports
    if not revision:
      query = PostsubmitReport.query(
          PostsubmitReport.gitiles_commit.project == project,
          PostsubmitReport.gitiles_commit.server_host == host,
          PostsubmitReport.bucket == bucket,
          PostsubmitReport.builder == builder, PostsubmitReport.visible == True,
          PostsubmitReport.modifier_id == modifier_id).order(
              -PostsubmitReport.commit_timestamp)
      entities = query.fetch(limit=1)
      report = entities[0]
      revision = report.gitiles_commit.revision
      ref = report.gitiles_commit.ref
      manifest = report.manifest
    else:
      report = PostsubmitReport.Get(
          server_host=host,
          project=project,
          ref=ref,
          revision=revision,
          bucket=bucket,
          builder=builder,
          modifier_id=modifier_id)
      if not report:
        return BaseHandler.CreateError('Report record not found', 404)
      else:
        manifest = report.manifest

    def _GetDataType(path):
      if not path or path.endswith('/'):
        return 'dirs'
      elif '>' in path:
        return 'components'
      else:
        return 'files'
    data_type = _GetDataType(path)
    if data_type == 'dirs':
      default_path = '//'
      template = 'coverage/summary_view.html'
    elif data_type == 'components':
      default_path = '>>'
      template = 'coverage/summary_view.html'
    else:
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
          builder=builder,
          modifier_id=modifier_id)
      if not entity:
        warning = (
            'File "%s" does not exist in this report, defaulting to root' %
            path)
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
          builder=builder,
          modifier_id=modifier_id)
      if not entity:
        warning = (
            'Path "%s" does not exist in this report, defaulting to root' %
            path)
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
            builder=builder,
            modifier_id=modifier_id)

    def _GetLineToData(manifest, path, metadata):
      """Returns coverage data per line in a file.

      Returns a list of tuples, sorted by line number, where the first
      element of tuple represents the line number and the second element is a
      dict with following structure

      {
        'line': (str) Contains the whole text of the line.
        'count': (int) Execution count of the line.
        'regions': (list) Present only if a line is partially covered. This is
                the output of _SplitLineIntoRegions().
        'is_partially_covered': (bool) True if a line can be split into
                regions of covered/uncovered blocks. False otherwise.
      }

      """
      line_to_data = collections.defaultdict(dict)
      if metadata.get('revision', ''):
        gs_path = utils.ComposeSourceFileGsPath(manifest, path,
                                                metadata['revision'])
        file_content = utils.GetFileContentFromGs(gs_path)
        if not file_content:
          # Fetching files from Gitiles is slow, only use it as a backup.
          file_content = utils.GetFileContentFromGitiles(
              manifest, path, metadata['revision'])
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
        return line_to_data

    data = {
        'metadata': entity.data,
    }
    if data_type == 'files':
      data['line_to_data'] = _GetLineToData(manifest, path, entity.data)

    # Compute the mapping of the name->path mappings in order.
    path_parts = _GetNameToPathSeparator(path, data_type)
    path_root, _ = _GetPathRootAndSeparatorFromDataType(data_type)
    metrics = code_coverage_util.GetMetricsBasedOnCoverageTool(
        coverage_tool=utils.GetPostsubmitPlatformInfoMap(luci_project)[platform]
        ['coverage_tool'])
    if modifier_id != 0:
      # Only line coverage metric is supported for cases other than
      # default post submit report
      metrics = [x for x in metrics if x['name'] == 'line']

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
                utils.GetPostsubmitPlatformInfoMap(luci_project)[platform]
                ['ui_name'],
            'path_root':
                path_root,
            'metrics':
                metrics,
            'data':
                data,
            'data_type':
                data_type,
            'test_suite_type':
                test_suite_type,
            'modifier_id':
                modifier_id,
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
