# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import json
import re

from recipe_engine.recipe_api import Property

PYTHON_VERSION_COMPATIBILITY = "PY3"

DEPS = [
    'recipe_engine/buildbucket',
    'recipe_engine/context',
    'recipe_engine/file',
    'recipe_engine/json',
    'recipe_engine/path',
    'recipe_engine/properties',
    'recipe_engine/raw_io',
    'recipe_engine/step',
    'depot_tools/gclient',
    'depot_tools/git',
    'depot_tools/gitiles',
]

PROPERTIES = {
    'source_repo':
        Property(help='The URL of the repo to be mirrored with submodules'),
    'target_repo':
        Property(help='The URL of the mirror repo to be built/maintained'),
    'extra_submodules':
        Property(
            default=[],
            help='A list of <path>=<url> strings, indicating extra submodules '
            'to add to the mirror repo.'),
    'refs':
        Property(
            default=['refs/heads/main'],
            help='A list of refs for which submodules should be added.'),
    'overlays':
        Property(
            default=[],
            help='A list of DEPS prefixes to be included as submodules to the '
            'mirror repo. Prefix is stripped from DEPS path. Example: '
            'if DEPS contains foo/bar=baz.git and overlay is foo, then the '
            'mirror repo will have submodule baz.git in path bar.'),
    'internal':
        Property(default="", help="If should fetch internal source"),
    'with_tags':
        Property(default=True, help='Whether to clone, fetch and push tags.'),
}

COMMIT_USERNAME = 'Submodules bot'
COMMIT_EMAIL_ADDRESS = \
    'infra-codesearch@chops-service-accounts.iam.gserviceaccount.com'

SHA1_RE = re.compile(r'[0-9a-fA-F]{40}')


def RunSteps(api, source_repo, target_repo, extra_submodules, refs, overlays,
             internal, with_tags):
  _, source_project = api.gitiles.parse_repo_url(source_repo)

  # NOTE: This name must match the definition in cr-buildbucket.cfg. Do not
  # change without adjusting that config to match.
  checkout_dir = api.m.path['cache'].join('codesearch_update_submodules_mirror')
  api.m.file.ensure_directory('Create checkout parent dir', checkout_dir)

  # We assume here that we won't have a mirror for two repos with the same name.
  # If we do, the directories will have the same name. This shouldn't be an
  # issue, but if it is we should add an intermediate directory with an
  # unambiguous name.
  #
  # We want to keep the final component equal to the below, as gclient/DEPS can
  # be sensitive to the name of the directory a repo is checked out to.
  #
  # The slash on the end doesn't make a difference for source_checkout_dir. But
  # it's necessary for the other uses for source_checkout_name, below.
  source_checkout_name = source_project[source_project.rfind('/') + 1:] + '/'
  source_checkout_dir = checkout_dir.join(source_checkout_name)

  # TODO: less hacky way of checking if the dir exists?
  glob = api.m.file.glob_paths('Check for existing source checkout dir',
                               checkout_dir, source_checkout_name)
  if not glob:
    # We don't depend on any particular cwd, as source_checkout_dir is absolute.
    # But we must supply *some* valid path, or it will fail to spawn the
    # process.
    with api.context(cwd=checkout_dir):
      # Don't use --no-tags even if with_tags is False here since clones may
      # time out.
      api.git('clone', source_repo, source_checkout_dir)

  # This is implicitly used as the cwd by all the git steps below.
  api.m.path['checkout'] = source_checkout_dir

  try:
    api.git('fetch', '-t' if with_tags else '-n')
    for ref in refs:
      if not ref.startswith('refs/heads'):
        api.git('fetch', 'origin', ref + ':' + RefToRemoteRef(ref))
  except api.step.StepFailure:
    # Remove broken source checkout so that subsequent runs don't fail because
    # of it.
    with api.context(cwd=checkout_dir):
      api.m.file.rmtree('Remove broken source checkout clone',
                        api.path.abspath(source_checkout_dir))
    raise

  for ref in refs:
    with api.step.nest('Process ' + ref):
      api.git('checkout', RefToRemoteRef(ref))

      gclient_spec = [{
          'managed': False,
          'name': source_checkout_name,
          'url': source_repo,
          'deps_file': 'DEPS'
      }]
      if internal:
        gclient_spec[0]['custom_vars'] = {'checkout_src_internal': True}
      gclient_spec_repr = ("solutions=" + repr(gclient_spec))
      with api.context(cwd=checkout_dir):
        deps = json.loads(
            api.gclient(
                'evaluate DEPS', [
                    'revinfo', '--deps', 'all', '--ignore-dep-type=cipd',
                    '--spec', gclient_spec_repr, '--output-json=-'
                ],
                stdout=api.raw_io.output_text()).stdout)

      for item in extra_submodules:
        path, url = item.split('=')
        deps[path] = {'url': url, 'rev': 'main'}

      update_index_entries, gitmodules_entries = GetSubmodules(
          api, deps, source_checkout_name, overlays)

      # This adds submodule entries to the index without cloning the underlying
      # repository.
      api.git(
          'update-index', '--add', *update_index_entries, name='Add gitlinks')

      api.file.write_text(
          'Write .gitmodules file', source_checkout_dir.join('.gitmodules'),
          '\n'.join(gitmodules_entries))
      api.git('add', '.gitmodules')

      api.git(
          '-c', 'user.name=%s' % COMMIT_USERNAME,
          '-c', 'user.email=%s' % COMMIT_EMAIL_ADDRESS,
          'commit', '-m', 'Synthetic commit for submodules',
          name='git commit')
      api.git(
          'push',
          '-o',
          'nokeycheck',
          target_repo,
          'HEAD:' + ref,
          # skip-validation is necessary as without it we cannot push >=10k
          # commits at once.
          '--push-option=skip-validation',
          # We've effectively deleted the commit that was at HEAD before. This
          # means that we've diverged from the remote repo, and hence must do a
          # force push.
          '--force',
          name='git push ' + ref)

  api.git('push', '-o', 'nokeycheck', target_repo,
          'refs/remotes/origin/main:refs/heads/main-original')

  if with_tags:
    # You can't use --all and --tags at the same time for some reason.
    # --mirror pushes both, but it also pushes remotes, which we don't want.
    api.git(
        'push',
        '-o',
        'nokeycheck',
        '--tags',
        target_repo,
        name='git push --tags')


def RefToRemoteRef(ref):
  ref = ref.replace('refs/heads', 'refs/remotes/origin')
  ref = ref.replace('refs/branch-heads', 'refs/remotes/branch-heads')
  return ref


def GetSubmodules(api, deps, source_checkout_name, overlays):
  gitmodules_entries = []
  update_index_entries = []
  for path, entry in deps.items():
    url = entry['url']
    rev = entry['rev']
    if rev is None:
      rev = 'HEAD'

    # Filter out the root repo itself, which shows up for some reason.
    if path == source_checkout_name:
      continue
    # Filter out deps that are nested within other deps. Submodules can't
    # represent this.
    if any(path != other_path and path.startswith(other_path + '/')
           for other_path in deps.keys()):
      continue

    # Filter out any DEPS that point outside of the repo, as there's no way to
    # represent this with submodules, unless they are in overlays.
    #
    # Note that source_checkout_name has a slash on the end, so this will
    # correctly filter out any path which has the checkout name as a prefix.
    # For example, src-internal in the src DEPS file.
    if not path.startswith(source_checkout_name):
      should_include = False
      for overlay in overlays:
        if path.startswith(overlay) and path != overlay:
          should_include = True
          path = str(path[len(overlay.rstrip('/')) + 1:])
          break
      if not should_include:
        continue
    else:
      # json.loads returns unicode but the recipe framework can only handle str.
      path = str(path[len(source_checkout_name):])

    path = path.rstrip('/')

    if not SHA1_RE.match(rev):
      if rev.startswith('origin/'):
        rev = rev[len('origin/'):]
      rev = api.git(
          'ls-remote', url, rev,
          stdout=api.raw_io.output_text()).stdout.split()[0]

    update_index_entries.extend(['--cacheinfo', '160000,%s,%s' % (rev, path)])

    gitmodules_entries.append('[submodule "%s"]\n\tpath = %s\n\turl = %s'
                              % (path, path, str(url)))

  return update_index_entries, gitmodules_entries


fake_src_deps = """
{
  "src/v8": {
    "url": "https://chromium.googlesource.com/v8/v8.git",
    "rev": "4ad2459561d76217c9b7aff412c5c086b491078a"
  },
  "src/buildtools": {
    "url": "https://chromium.googlesource.com/chromium/buildtools.git",
    "rev": "13a00f110ef910a25763346d6538b60f12845656"
  },
  "src-internal": {
    "url": "https://chrome-internal.googlesource.com/chrome/src-internal.git",
    "rev": "34b7d6a218430e7ff716b81854743a30cfbd3967"
  },
  "tooling/some_tooling_repo": {
    "url": "https://chrome-internal.googlesource.com/some_tooling_repo.git",
    "rev": "0000000000000000000000000000000000000000"
  },
  "src/": {
    "url": "https://chromium.googlesource.com/chromium/src.git",
    "rev": null
  }
}
"""

fake_deps_with_symbolic_ref = """
{
  "src/v8": {
    "url": "https://chromium.googlesource.com/v8/v8.git",
    "rev": "origin/main"
  }
}
"""

fake_deps_with_nested_dep = """
{
  "src/third_party/gsutil": {
    "url": "https://chromium.googlesource.com/external/gsutil/src.git",
    "rev": "5cba434b828da428a906c8197a23c9ae120d2636"
  },
  "src/third_party/gsutil/boto": {
    "url": "https://chromium.googlesource.com/external/boto.git",
    "rev": "98fc59a5896f4ea990a4d527548204fed8f06c64"
  }
}
"""

fake_deps_with_trailing_slash = """
{
  "src/v8/": {
    "url": "https://chromium.googlesource.com/v8/v8.git",
    "rev": "4ad2459561d76217c9b7aff412c5c086b491078a"
  }
}
"""

def GenTests(api):
  yield (api.test('first_time_running') + api.properties(
      source_repo='https://chromium.googlesource.com/chromium/src',
      target_repo='https://chromium.googlesource.com/codesearch/src_mirror') +
         api.step_data(
             'Check for existing source checkout dir',
             # Checkout doesn't exist.
             api.raw_io.stream_output_text('', stream='stdout')) +
         api.step_data(
             'Process refs/heads/main.gclient evaluate DEPS',
             api.raw_io.stream_output_text(fake_src_deps, stream='stdout')))

  yield (api.test('existing_checkout_git_dir') + api.properties(
      source_repo='https://chromium.googlesource.com/chromium/src',
      target_repo='https://chromium.googlesource.com/codesearch/src_mirror') +
         api.step_data('Check for existing source checkout dir',
                       api.raw_io.stream_output_text('src', stream='stdout')) +
         api.step_data('git fetch', retcode=128))

  yield (api.test('existing_checkout_new_commits') + api.properties(
      source_repo='https://chromium.googlesource.com/chromium/src',
      target_repo='https://chromium.googlesource.com/codesearch/src_mirror') +
         api.step_data('Check for existing source checkout dir',
                       api.raw_io.stream_output_text('src', stream='stdout')) +
         api.step_data(
             'Process refs/heads/main.'
             'gclient evaluate DEPS',
             api.raw_io.stream_output_text(fake_src_deps, stream='stdout')))

  yield (
      api.test('existing_checkout_latest_commit_not_by_bot') + api.properties(
          source_repo='https://chromium.googlesource.com/chromium/src',
          target_repo='https://chromium.googlesource.com/codesearch/src_mirror')
      + api.step_data('Check for existing source checkout dir',
                      api.raw_io.stream_output_text('src', stream='stdout')) +
      api.step_data(
          'Process refs/heads/main.gclient evaluate DEPS',
          api.raw_io.stream_output_text(fake_src_deps, stream='stdout')))

  yield (api.test('ref_that_needs_resolving') + api.properties(
      source_repo='https://chromium.googlesource.com/chromium/src',
      target_repo='https://chromium.googlesource.com/codesearch/src_mirror') +
         api.step_data('Check for existing source checkout dir',
                       api.raw_io.stream_output_text('src', stream='stdout')) +
         api.step_data(
             'Process refs/heads/main.gclient evaluate DEPS',
             api.raw_io.stream_output_text(
                 fake_deps_with_symbolic_ref, stream='stdout')) +
         api.step_data(
             'Process refs/heads/main.git ls-remote',
             api.raw_io.stream_output_text(
                 '91c13923c1d136dc688527fa39583ef61a3277f7\trefs/heads/main',
                 stream='stdout')))

  yield (api.test('nested_deps') + api.properties(
      source_repo='https://chromium.googlesource.com/chromium/src',
      target_repo='https://chromium.googlesource.com/codesearch/src_mirror') +
         api.step_data('Check for existing source checkout dir',
                       api.raw_io.stream_output_text('src', stream='stdout')) +
         api.step_data(
             'Process refs/heads/main.gclient evaluate DEPS',
             api.raw_io.stream_output_text(
                 fake_deps_with_nested_dep, stream='stdout')))

  yield (api.test('trailing_slash') + api.properties(
      source_repo='https://chromium.googlesource.com/chromium/src',
      target_repo='https://chromium.googlesource.com/codesearch/src_mirror') +
         api.step_data('Check for existing source checkout dir',
                       api.raw_io.stream_output_text('src', stream='stdout')) +
         api.step_data(
             'Process refs/heads/main.gclient evaluate DEPS',
             api.raw_io.stream_output_text(
                 fake_deps_with_trailing_slash, stream='stdout')))

  yield (api.test('extra_submodule') + api.properties(
      source_repo='https://chromium.googlesource.com/chromium/src',
      target_repo='https://chromium.googlesource.com/codesearch/src_mirror',
      extra_submodules=['src/extra=https://extra.googlesource.com/extra']) +
         api.step_data('Check for existing source checkout dir',
                       api.raw_io.stream_output_text('src', stream='stdout')) +
         api.step_data(
             'Process refs/heads/main.gclient evaluate DEPS',
             api.raw_io.stream_output_text(fake_src_deps, stream='stdout')) +
         api.step_data(
             'Process refs/heads/main.git ls-remote',
             api.raw_io.stream_output_text(
                 'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\trefs/heads/main',
                 stream='stdout')))

  yield (api.test('extra_branches') + api.properties(
      source_repo='https://chromium.googlesource.com/chromium/src',
      target_repo='https://chromium.googlesource.com/codesearch/src_mirror',
      refs=['refs/heads/main', 'refs/branch-heads/4044'],
  ) + api.step_data(
      'Check for existing source checkout dir',
      # Checkout doesn't exist.
      api.raw_io.stream_output_text('', stream='stdout')) + api.step_data(
          'Process refs/heads/main.gclient evaluate DEPS',
          api.raw_io.stream_output_text(fake_src_deps, stream='stdout')) +
         api.step_data(
             'Process refs/branch-heads/4044.gclient evaluate DEPS',
             api.raw_io.stream_output_text(fake_src_deps, stream='stdout')))

  yield (api.test('overlays') + api.properties(
      source_repo='https://chromium.googlesource.com/chromium/src',
      target_repo='https://chromium.googlesource.com/codesearch/src_mirror',
      overlays=['tooling']) +
         api.step_data('Check for existing source checkout dir',
                       api.raw_io.stream_output_text('src', stream='stdout')) +
         api.step_data(
             'Process refs/heads/main.gclient evaluate DEPS',
             api.raw_io.stream_output_text(fake_src_deps, stream='stdout')))

  yield (api.test('fetch_internal') + api.properties(
      source_repo='https://chromium.googlesource.com/chromium/src',
      target_repo='https://chromium.googlesource.com/codesearch/src_mirror',
      internal="true") + api.step_data(
          'Check for existing source checkout dir',
          # Checkout doesn't exist.
          api.raw_io.stream_output_text('', stream='stdout')) + api.step_data(
              'Process refs/heads/main.gclient evaluate DEPS',
              api.raw_io.stream_output_text(fake_src_deps, stream='stdout')))

  yield (api.test('with_tags_false') + api.properties(
      source_repo='https://chromium.googlesource.com/chromium/src',
      target_repo='https://chromium.googlesource.com/codesearch/src_mirror',
      internal='true',
      with_tags=False) +
         api.step_data('Check for existing source checkout dir',
                       api.raw_io.stream_output_text('', stream='stdout')) +
         api.step_data(
             'Process refs/heads/main.gclient evaluate DEPS',
             api.raw_io.stream_output_text(fake_src_deps, stream='stdout')))
