# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from PB.recipes.infra import images_pins_roller as images_pins_roller_pb

DEPS = [
    'depot_tools/git',
    'depot_tools/git_cl',

    'recipe_engine/context',
    'recipe_engine/json',
    'recipe_engine/path',
    'recipe_engine/properties',

    'cloudbuildhelper',
    'infra_checkout',
]


PROPERTIES = images_pins_roller_pb.Inputs


def RunSteps(api, properties):
  # Use 'cloudbuildhelper' that comes with the infra checkout (it's in PATH),
  # to make sure builders use same version as developers.
  co = api.infra_checkout.checkout(gclient_config_name='infra', internal=False)
  co.gclient_runhooks()

  # Checkout the repo with the file to be updated.
  root = api.path['cache'].join('builder', 'checkout')
  api.git.checkout(properties.repo_url, dir_path=root, submodules=False)

  with co.go_env():
    # Use 'cloudbuildhelper' which is in PATH now.
    api.cloudbuildhelper.command = 'cloudbuildhelper'
    api.cloudbuildhelper.report_version()
    # Modify pins_yaml file in-place.
    updated = api.cloudbuildhelper.update_pins(root.join(properties.pins_yaml))

  # Send a CL if something has changed.
  if updated:
    _send_cl(api, root, updated, properties.tbr, properties.commit)


def _send_cl(api, root, updated, tbrs, commit):
  desc = [
      'Roll pinned docker image tags.',
      '',
      'Updated pins:',
  ] + ['  * %s' % p for p in updated]
  desc = '\n'.join(desc)

  with api.context(cwd=root):
    api.git('branch', '-D', 'roll-pins', ok_ret=(0, 1))
    api.git('checkout', '-t', 'origin/master', '-b', 'roll-pins')
    api.git('commit', '-a', '-m', desc)

    api.git_cl.upload(desc, name='git cl upload', upload_args=[
        '--bypass-hooks',  # don't have full checkout to run hooks locally
        '--force',         # skip asking for description, we already set it
    ] + [
        '--tbrs=%s' % tbr for tbr in tbrs
    ] + (['--use-commit-queue'] if commit else []))

    # Put a link to the uploaded CL.
    issue_step = api.git_cl(
        'issue', ['--json', api.json.output()],
        name='git cl issue',
        step_test_data=lambda: api.json.test_api.output({
            'issue': 123456789,
            'issue_url': 'https://chromium-review.googlesource.com/c/123456789',
        }),
    )
    out = issue_step.json.output
    issue_step.presentation.links['Issue %s' % out['issue']] = out['issue_url']


def GenTests(api):
  yield (
      api.test('rolls') +
      api.properties(
          images_pins_roller_pb.Inputs(
              repo_url='https://chromium.googlesource.com/infra/infra',
              pins_yaml='build/images/pins.yaml',
              tbr=['a@example.com', 'b@example.com'],
              commit=True,
          )
      )
  )
