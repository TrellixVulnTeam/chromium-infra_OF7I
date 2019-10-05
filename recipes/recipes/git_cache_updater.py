# Copyright 2014 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Updates the Git Cache zip files."""

import re

from recipe_engine import recipe_api
from recipe_engine import post_process
from PB.recipe_engine import result as result_pb
from PB.go.chromium.org.luci.buildbucket.proto import common as bb_common_pb

from PB.recipes.infra import git_cache_updater as git_cache_updater_pb


DEPS = [
  'recipe_engine/buildbucket',
  'recipe_engine/context',
  'recipe_engine/file',
  'recipe_engine/raw_io',
  'recipe_engine/futures',
  'recipe_engine/path',
  'recipe_engine/properties',
  'recipe_engine/runtime',
  'recipe_engine/step',
  'recipe_engine/url',

  'depot_tools/depot_tools',
  'depot_tools/git',
]

PROPERTIES = git_cache_updater_pb.Inputs

OK, EMPTY, NO_MASTER = range(3)


def _list_host_repos(api, host_url):
  host_url = host_url.rstrip('/')
  with api.depot_tools.on_path():
    output = api.url.get_text('%s/?format=TEXT' % host_url,
                              default_test_data=TEST_REPOS).output
    return output.strip().splitlines()


def _repos_to_urls(host_url, repos):
  host_url = host_url.rstrip('/')
  return ['%s/%s' % (host_url, repo) for repo in repos]


class _InvalidInput(Exception):
  pass


def _get_repo_urls(api, inputs):
  if inputs.git_host.host:
    assert not inputs.repo_urls, 'only 1 of (git_host, repo_urls) allowed'
    repos = _list_host_repos(api, 'https://' + inputs.git_host.host)
    if inputs.git_host.exclude_repos:
      exclude_regexps = []
      for i, r in enumerate(inputs.git_host.exclude_repos):
        try:
          exclude_regexps.append(re.compile('^' + r + '$', re.IGNORECASE))
        except Exception as e:
          raise _InvalidInput(
              'invalid regular expression[%d] %r: %s' % (i, r, e))
      repos = [repo for repo in repos
               if all(not r.match(repo) for r in exclude_regexps)]
    return _repos_to_urls('https://' + inputs.git_host.host, repos)

  if inputs.repo_urls:
    return list(inputs.repo_urls)

  raise _InvalidInput('repo_urls or git_host.host must be provided')


def _do_update_bootstrap(api, url, work_dir, gc_aggressive):
  opts = [
    '--cache-dir', work_dir,
    '--verbose',
    url,
  ]

  with api.step.nest(url) as summary:
    api.step(
        name='populate',
        cmd=[
          'git_cache.py', 'populate',
          '--reset-fetch-config',

          # By default, "refs/heads/*" and refs/tags/* are checked out by
          # git_cache. However, for heavy branching repos,
          # 'refs/branch-heads/*' is also very useful (crbug/942169).
          # This is a noop for repos without refs/branch-heads.
          '--ref', 'refs/branch-heads/*',

          '--break-locks',
        ]+opts,
        cost=api.step.ResourceCost(disk=20))

    repo_path = api.path.abs_to_path(api.step(
        name='lookup repo_path',
        cmd=['git_cache.py', 'exists'] + opts,
        stdout=api.raw_io.output(),
        step_test_data=lambda: api.raw_io.test_api.stream_output(
            api.path.join(work_dir, url.strip('https://'))+'\n',
        ),
    ).stdout.strip())

    with api.context(cwd=repo_path):
      stats = api.git.count_objects(
          can_fail_build=True,
          # TODO(iannucci): ugh, the test mock for this is horrendous.
          #   1) it should default to something automatically
          #   2) test_api.count_objects_output should return a TestData, not
          #      a string.
          step_test_data=lambda: api.raw_io.test_api.stream_output(
              api.git.test_api.count_objects_output(10)))

      # Scale the memory cost of this update by size-pack raised to 1.5. This is
      # an arbitrary scaling factor, but it allows multiple small repos to run
      # in parallel but allows large repos (e.g. chromium) to exclusively use
      # all the memory on the system.
      mem_cost = int((stats['size'] + stats['size-pack']) ** 1.5)
      if mem_cost == 0:
        # some repos can be empty (e.g. they're an "ACL-only" repo), and
        # update-bootstrap doesn't like that, so skip them.
        api.step('repo is empty; skipping update', cmd=None)
        summary.step_text = "[empty]"
        summary.status = api.step.FAILURE  # TODO(iannucci): warning
        return EMPTY

      if api.git('rev-parse', '-q', '--verify', 'master', ok_ret='any').retcode:
        api.step('repo has no master ref; skipping update', cmd=None)
        summary.step_text = "[no master ref]"
        summary.status = api.step.FAILURE  # TODO(iannucci): warning
        return NO_MASTER

    gc_aggressive_opt = []
    if gc_aggressive:
      gc_aggressive_opt = ['--gc-aggressive']

    api.step(
        name='update bootstrap',
        cmd=[
          'git_cache.py', 'update-bootstrap',
          '--skip-populate', '--prune',
        ] + opts + gc_aggressive_opt,
        cost=api.step.ResourceCost(memory=mem_cost, net=10))

    summary.step_text = "[ok]"
    return OK


def RunSteps(api, inputs):
  try:
    repo_urls = _get_repo_urls(api, inputs)
  except _InvalidInput as e:
    return result_pb.RawResult(
        status=bb_common_pb.FAILURE,
        summary_markdown=e.message)

  work_dir = api.path['cache'].join('builder', 'w')
  api.file.ensure_directory('ensure work_dir', work_dir)

  env = {
    # Turn off the low speed limit, since checkout will be long.
    'GIT_HTTP_LOW_SPEED_LIMIT': '0',
    'GIT_HTTP_LOW_SPEED_TIME': '0',
    # Ensure git-number tool can be used.
    'CHROME_HEADLESS': '1',
  }
  if api.runtime.is_experimental:
    assert inputs.override_bucket, 'override_bucket required for experiments'
  if inputs.override_bucket:
    env['OVERRIDE_BOOTSTRAP_BUCKET'] = inputs.override_bucket

  work = []
  with api.context(env=env), api.depot_tools.on_path():
    for url in sorted(repo_urls):
      work.append(api.futures.spawn_immediate(
          _do_update_bootstrap, api, url, work_dir, inputs.gc_aggressive,
          __name=url))

  total = len(work)
  success = warning = 0
  failed_repos = []
  empties = masterless = 0
  for future in api.futures.iwait(work):
    try:
      status = future.result()
    except Exception:  # pylint: disable=broad-except
      failed_repos.append(future.name)
      continue

    if status == OK:
      success += 1
    elif status == EMPTY:
      empties += 1
      warning += 1
    elif status == NO_MASTER:
      masterless += 1
      warning += 1
    else:
      assert False, 'unknown status %r' % (status,)  # pragma: no cover

  status = bb_common_pb.FAILURE if failed_repos else bb_common_pb.SUCCESS
  summary = 'Updated cache for %d/%d repos.' % (success, total)
  if warning:
    summary += '\n\nEncountered warnings for %d repos:' % (warning,)
    if empties:
      summary += '\n  * empty (repo has no objects): %d' % (empties,)
    if masterless:
      summary += '\n  * no master ref: %d' % (masterless,)
  if failed_repos:
    summary += '\n\nEncountered failures for %d repos:' % (len(failed_repos),)
    for repo_name in failed_repos:
      summary += '\n  * ' + repo_name

  return result_pb.RawResult(status=status, summary_markdown=summary)


TEST_REPOS = """
All-Projects
All-Users
apps
chromium/src
foo/bar
"""


def GenTests(api):
  yield (
      api.test('needs input')
      + api.post_process(post_process.StatusFailure)
      + api.post_process(post_process.DropExpectation)
  )

  yield (
      api.test('one-repo-experiment-aggressive')
      + api.runtime(is_experimental=True, is_luci=True)
      + api.properties(git_cache_updater_pb.Inputs(
          override_bucket='experimental-gs-bucket',
          repo_urls=['https://chromium.googlesource.com/v8/v8'],
          gc_aggressive=True,
      ))
  )

  yield (
      api.test('one-repo-empty')
      + api.runtime(is_experimental=True, is_luci=True)
      + api.properties(git_cache_updater_pb.Inputs(
          override_bucket='experimental-gs-bucket',
          repo_urls=['https://chromium.googlesource.com/empty'],
          gc_aggressive=True,
      ))
      + api.override_step_data(
          'https://chromium.googlesource.com/empty.git count-objects',
          api.raw_io.stream_output(api.git.count_objects_output(0)),
      )
  )

  yield (
      api.test('one-repo-fail')
      + api.runtime(is_experimental=True, is_luci=True)
      + api.properties(git_cache_updater_pb.Inputs(
          override_bucket='experimental-gs-bucket',
          repo_urls=['https://chromium.googlesource.com/fail'],
          gc_aggressive=True,
      ))
      + api.override_step_data(
          'https://chromium.googlesource.com/fail.populate',
          retcode=1,
      )
  )

  yield (
      api.test('one-repo-no-master')
      + api.runtime(is_experimental=True, is_luci=True)
      + api.properties(git_cache_updater_pb.Inputs(
          override_bucket='experimental-gs-bucket',
          repo_urls=['https://chromium.googlesource.com/bogus'],
          gc_aggressive=True,
      ))
      + api.override_step_data(
          'https://chromium.googlesource.com/bogus.git rev-parse',
          retcode=1,
      )
  )


  yield (
      api.test('host-with-exclusions')
      + api.properties(git_cache_updater_pb.Inputs(
          git_host=git_cache_updater_pb.Inputs.GitHost(
              host='chromium.googlesource.com',
              exclude_repos=[
                'foo/.+',
                'all-projects',
                'all-users',
              ],
          ),
      ))
  )

  yield (
      api.test('host-with-incorrect-regexp-exclude')
      + api.properties(git_cache_updater_pb.Inputs(
          git_host=git_cache_updater_pb.Inputs.GitHost(
              host='chromium.googlesource.com',
              exclude_repos=[
                '?.\\',
              ],
          ),
      ))
      + api.post_process(post_process.StatusFailure)
      + api.post_process(post_process.DropExpectation)
  )
