# Copyright 2015 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import textwrap

from recipe_engine.recipe_api import Property

DEPS = [
    'depot_tools/osx_sdk',
    'infra_checkout',
    'recipe_engine/buildbucket',
    'recipe_engine/cipd',
    'recipe_engine/context',
    'recipe_engine/json',
    'recipe_engine/path',
    'recipe_engine/platform',
    'recipe_engine/properties',
    'recipe_engine/raw_io',
    'recipe_engine/step',
    'recipe_engine/tricium',
]

PROPERTIES = {
    'GOARCH':
        Property(
            default=None,
            kind=str,
            help='Set GOARCH environment variable for go build+test'),
    'go_version_variant':
        Property(
            default=None,
            kind=str,
            help='A go version variant to bootstrap, see bootstrap.py'),
    'go_modules':
        Property(
            default=False, kind=bool, help='Whether to run in Go Modules mode'),
    'run_integration_tests':
        Property(
            default=False, kind=bool, help='Whether to run integration tests'),
    'run_lint':
        Property(default=False, kind=bool, help='Whether to run linter'),
}

LUCI_GO_PATH_IN_INFRA = 'infra/go/src/go.chromium.org/luci'


def apply_golangci_lint(api, co):
  go_files = sorted(
      set([
          api.path.dirname(f) + "/..."
          for f in co.get_changed_files()
          if f.endswith('.go')
      ]))

  if not go_files:
    return  # pragma: no cover

  linter = api.cipd.ensure_tool('infra/3pp/tools/golangci-lint/${platform}',
                                'version:2@1.40.0')
  result = api.step(
      'run golangci-lint',
      [
          linter,
          'run',
          '--out-format=json',
          '--issues-exit-code=0',
          '--timeout=5m',
      ] + go_files,
      step_test_data=lambda: api.json.test_api.output_stream({
          "Issues": [{
              "FromLinter": "deadcode",
              "Text": "`foo` is unused",
              "Severity": "",
              "SourceLines": ["func foo() {}"],
              "Pos": {
                  "Filename": "client/cmd/isolate/lib/batch_archive.go",
                  "Offset": 7960,
                  "Line": 250,
                  "Column": 6
              },
              "HunkPos": 4,
              "ExpectedNoLintLinter": ""
          }],
      }),
      stdout=api.json.output())

  for issue in result.stdout.get("Issues") or ():
    pos = issue["Pos"]
    line = pos["Line"]
    api.tricium.add_comment("golangci-lint (%s)" % issue["FromLinter"],
                            issue["Text"], pos["Filename"], line, line + 1,
                            pos["Column"] - 1, 0)

  api.tricium.write_comments()


def RunSteps(
    api, GOARCH, go_version_variant, go_modules,
    run_integration_tests, run_lint):
  co = api.infra_checkout.checkout(
      'luci_go',
      patch_root=LUCI_GO_PATH_IN_INFRA,
      go_version_variant=go_version_variant,
      go_modules=go_modules)
  is_presubmit = 'presubmit' in api.buildbucket.builder_name.lower()
  if is_presubmit or run_lint:
    co.commit_change()
  co.gclient_runhooks()

  env = {}
  if GOARCH is not None:
    env['GOARCH'] = GOARCH
  if run_integration_tests:
    env['INTEGRATION_TESTS'] = '1'
    # Flag to running spanner tests using the Cloud Spanner Emulator.
    # TODO(crbug.com/1066993): Remove this extra flag when we're ready to
    # always running spanner tests using the emulator.
    env['SPANNER_EMULATOR'] = '1'

  with api.context(env=env), api.osx_sdk('mac'), co.go_env():
    if is_presubmit:
      co.run_presubmit()
    else:
      luci_go = co.path.join('infra', 'go', 'src', 'go.chromium.org', 'luci')
      with api.context(cwd=luci_go):
        if run_lint:
          apply_golangci_lint(api, co)
        else:
          api.step('go build', ['go', 'build', './...'])
          api.step('go test', ['go', 'test', './...'])
          if not api.platform.is_win:
            # Windows bots do not have gcc installed at the moment.
            api.step('go test -race', ['go', 'test', '-race', './...'])


def GenTests(api):
  for plat in ('linux', 'mac', 'win'):
    yield (
      api.test('luci_go_%s' % plat) +
      api.platform(plat, 64) +
      api.buildbucket.ci_build(
          'infra', 'ci', 'luci-gae-trusty-64',
          git_repo="https://chromium.googlesource.com/infra/infra",
          revision='1'*40) +
      # Sadly, hacks in gclient required to patch non-main git repo in
      # a solution requires revision as a property :(
      api.properties(revision='1'*40)
    )

  yield (api.test('presubmit_try_job') + api.buildbucket.try_build(
      'infra',
      'try',
      'Luci-go Presubmit',
      change_number=607472,
      patch_set=2,
  ) + api.step_data('presubmit', api.json.output([[]])))

  yield (api.test('lint_try_job') + api.buildbucket.try_build(
      'infra',
      'try',
      'luci-go lint',
      change_number=607472,
      patch_set=2,
  ) + api.properties(run_lint=True) + api.step_data(
      'get change list',
      stdout=api.raw_io.output(textwrap.dedent("""\
      client/cmd/isolate/lib/batch_archive.go
      client/cmd/isolate/lib/archive.go
      client/cmd/isolated/lib/archive.go
      """))))

  yield (
    api.test('override_GOARCH') +
    api.platform('linux', 64) +
    api.buildbucket.try_build(
        'infra', 'try', 'luci-go-trusty-64',
        git_repo='https://chromium.googlesource.com/infra/luci/luci-go',
        change_number=607472,
        patch_set=2,
    ) + api.properties(GOARCH='386')
  )

  yield (
    api.test('integration_tests') +
    api.buildbucket.try_build(
        'infra', 'try', 'integration_tests', change_number=607472, patch_set=2,
    ) +
    api.properties(run_integration_tests=True)
  )

  yield (
    api.test('override_go_version') +
    api.platform('linux', 64) +
    api.buildbucket.try_build(
        'infra', 'try', 'luci-go-trusty-64',
        git_repo='https://chromium.googlesource.com/infra/luci/luci-go',
        change_number=607472,
        patch_set=2,
    ) + api.properties(go_version_variant='bleeding_edge')
  )

  yield (
    api.test('go_modules') +
    api.platform('linux', 64) +
    api.buildbucket.try_build(
        'infra', 'try', 'luci-go-trusty-64',
        git_repo='https://chromium.googlesource.com/infra/luci/luci-go',
        change_number=607472,
        patch_set=2,
    ) + api.properties(go_modules=True)
  )
