# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from recipe_engine import post_process

from PB.recipes.infra import tricium_infra as tricium_infra_pb

DEPS = [
    'infra_checkout',
    'depot_tools/bot_update',
    'depot_tools/gclient',
    'depot_tools/gerrit',
    'depot_tools/tryserver',
    'recipe_engine/buildbucket',
    'recipe_engine/json',
    'recipe_engine/path',
    'recipe_engine/platform',
    'recipe_engine/properties',
    'recipe_engine/step',
    'recipe_engine/tricium',
]

PROPERTIES = tricium_infra_pb.Inputs


def RunSteps(api, inputs):
  """This recipe runs legacy analyzers for the infra repo."""
  assert api.platform.is_linux and api.platform.bits == 64

  if not inputs or not inputs.gclient_config_name:
    raise api.step.StepFailure('Input properties are required')

  # We want line numbers for the file as it is in the CL, not rebased.
  # gerrit_no_rebase_patch_ref prevents rebasing.
  checkout = api.infra_checkout.checkout(
      inputs.gclient_config_name,
      patch_root=inputs.patch_root,
      gerrit_no_rebase_patch_ref=True)
  checkout.gclient_runhooks()
  commit_message = api.gerrit.get_change_description(
      'https://%s' % api.tryserver.gerrit_change.host,
      api.tryserver.gerrit_change.change, api.tryserver.gerrit_change.patchset)
  input_dir = api.path['checkout']
  affected_files = [
      f for f in checkout.get_changed_files()
      if api.path.exists(input_dir.join(f)) and 'third_party/' not in f
  ]

  # TODO(qyearsley): Add a mapping to api.tricium.analyzers class itself, to
  # avoid having to access __dict___.
  analyzers = []
  analyzers_mapping = {}
  for member in api.tricium.analyzers.__dict__.values():
    if hasattr(member, 'name'):
      analyzers_mapping[member.name] = member
  analyzers = [analyzers_mapping[name] for name in inputs.analyzers]

  api.tricium.run_legacy(analyzers, input_dir, affected_files, commit_message)


def GenTests(api):

  def test_with_patch(name, affected_files):
    test = api.test(
        name,
        api.platform('linux', 64),
        api.buildbucket.try_build(
            project='infra',
            bucket='try',
            builder='tricium-infra',
            git_repo='https://chromium.googlesource.com/infra/infra') +
        api.override_step_data(
            'gerrit changes',
            api.json.output([{
                'revisions': {
                    'aaaa': {
                        '_number': 7,
                        'commit': {
                            'author': {
                                'email': 'user@a.com'
                            },
                            'message': 'my commit msg',
                        }
                    }
                }
            }])),
    )
    existing_files = [
        api.path['cache'].join('builder', x) for x in affected_files
    ]
    test += api.path.exists(*existing_files)
    return test

  yield (api.test('needs_input') +
         api.post_process(post_process.StatusFailure) +
         api.post_process(post_process.DropExpectation))

  yield (test_with_patch('infra_repo', ['README.md']) + api.properties(
      tricium_infra_pb.Inputs(
          gclient_config_name='infra',
          patch_root='infra',
          analyzers=['Copyright', 'ESlint', 'Gosec', 'Spacey', 'Spellchecker']))
         + api.post_process(post_process.DropExpectation))

  yield (test_with_patch('luci_py_repo', ['README.md']) + api.properties(
      tricium_infra_pb.Inputs(
          gclient_config_name='luci_py',
          patch_root='luci',
          analyzers=['Spacey', 'Spellchecker'])) +
         api.post_process(post_process.DropExpectation))

  yield (test_with_patch('luci_go_repo', ['README.md']) + api.properties(
      tricium_infra_pb.Inputs(
          gclient_config_name='luci_go',
          patch_root='infra',
          analyzers=['Gosec', 'Spacey', 'Spellchecker'])) +
         api.post_process(post_process.DropExpectation))
