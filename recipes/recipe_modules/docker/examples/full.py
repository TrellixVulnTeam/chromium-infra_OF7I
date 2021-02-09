# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from recipe_engine.post_process import DoesNotRun, Filter, StatusFailure

DEPS = [
    'recipe_engine/raw_io',
    'recipe_engine/step',
    'docker',
]


def RunSteps(api):
  api.docker.ensure_installed()
  version = api.docker.get_version()
  if version:
    api.step('log version', cmd=None).presentation.step_text = version
  api.docker.login()
  api.docker.pull('testimage')
  api.docker.run(
      'testimage',
      cmd_args=['test', 'cmd'],
      dir_mapping=[('/foo', '/bar')],
      env={
          'var1': '1',
          'var2': '2'
      },
      inherit_luci_context=True,
  )
  api.docker('push',
             'gcr.io/chromium-container-registry/image:2018-11-16-01-25')


def GenTests(api):
  yield api.test('example')

  yield api.test(
      'fail_installed',
      api.step_data('ensure docker installed', retcode=1),
      api.post_process(StatusFailure),
  )

  yield api.test(
      'fail_get_version',
      api.override_step_data('docker version',
                             api.raw_io.stream_output('Foo: bar')),
      api.post_process(DoesNotRun, 'log version'),
      api.post_process(Filter('docker version')),
  )
