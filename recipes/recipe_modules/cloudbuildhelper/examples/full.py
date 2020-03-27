# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

DEPS = [
  'cloudbuildhelper',
  'recipe_engine/json',
  'recipe_engine/step',
]


def RunSteps(api):
  api.cloudbuildhelper.report_version()

  # Coverage for api.cloudbuildhelper.build(...).
  build(api)
  # Coverage for api.cloudbuildhelper.upload(...).
  upload(api)

  # Use custom binary from this point onward, for test coverage.
  api.cloudbuildhelper.command = 'custom_cloudbuildhelper'
  # Updating pins.
  assert api.cloudbuildhelper.update_pins('some/pins.yaml')


def build(api):
  # Building with all args.
  img = api.cloudbuildhelper.build(
      manifest='some/dir/target.yaml',
      canonical_tag='123_456',
      build_id='bid',
      infra='dev',
      labels={'l1': 'v1', 'l2': 'v2'},
      tags=['latest', 'another'],
  )

  expected = api.cloudbuildhelper.Image(
      image='example.com/fake-registry/target',
      digest='sha256:34a04005bcaf206e...',
      tag='123_456',
      view_image_url='https://example.com/image/target',
      view_build_url='https://example.com/build/target')
  assert img == expected, img

  # With minimal args and custom emulated output.
  custom = api.cloudbuildhelper.Image(
      image='xxx',
      digest='yyy',
      tag=None,
      view_image_url='https://example.com/image/another',
      view_build_url='https://example.com/build/another')
  img = api.cloudbuildhelper.build('another.yaml', step_test_image=custom)
  assert img == custom, img

  # Using non-canonical tag.
  api.cloudbuildhelper.build('a.yaml', tags=['something'])

  # Using :inputs-hash canonical tag.
  api.cloudbuildhelper.build('b.yaml', canonical_tag=':inputs-hash')

  # Image that wasn't uploaded anywhere.
  img = api.cloudbuildhelper.build(
      'third.yaml', step_test_image=api.cloudbuildhelper.NotUploadedImage)
  assert img == api.cloudbuildhelper.NotUploadedImage, img

  # Possibly failing build.
  try:
    api.cloudbuildhelper.build('fail_maybe.yaml')
  except api.step.StepFailure:
    pass


def upload(api):
  # Passing all args.
  tarball = api.cloudbuildhelper.upload(
      manifest='some/dir/target.yaml',
      canonical_tag='123_456',
      build_id='bid',
      infra='dev',
  )
  expected = api.cloudbuildhelper.Tarball(
      name='example/target',
      bucket='example',
      path='tarballs/example/target/82ac16b294bc0f98....tar.gz',
      sha256='82ac16b294bc0f98...',
      version='123_456',
  )
  assert tarball == expected, tarball

  # With minimal args and custom emulated output.
  custom = api.cloudbuildhelper.Tarball(
      name='blah/target',
      bucket='some-bucket',
      path='some/path/file.tar.gz',
      sha256='111111...',
      version='4567-789',
  )
  tarball = api.cloudbuildhelper.upload(
      'another.yaml', canonical_tag='ignored', step_test_tarball=custom)
  assert tarball == custom, tarball

  # Possibly failing upload.
  try:
    api.cloudbuildhelper.upload('fail_maybe.yaml', canonical_tag='tag')
  except api.step.StepFailure:
    pass


def GenTests(api):
  yield api.test('simple')

  yield (
      api.test('failing') +
      api.step_data(
          'cloudbuildhelper build fail_maybe',
          api.cloudbuildhelper.build_error_output('Boom'),
          retcode=1,
      ) +
      api.step_data(
          'cloudbuildhelper upload fail_maybe',
          api.cloudbuildhelper.upload_error_output('Boom'),
          retcode=1
      )
  )
