# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

DEPS = [
  'provenance',
  'recipe_engine/path',
]


def RunSteps(api):
  api.provenance.generate(
    'projects/PROJECT/locations/global/keyRings/KEYRING/cryptoKeys/KEY',
    api.path['start_dir'].join('input.json'),
    api.path['cleanup'].join('output.attestation'),
  )
  # Generate another attestation; the module shouldn't install provenance again.
  api.provenance.generate(
    'projects/PROJECT/locations/global/keyRings/KEYRING/cryptoKeys/KEY',
    api.path['start_dir'].join('another-input.json'),
    api.path['cleanup'].join('another-output.attestation'),
  )


def GenTests(api):
  yield api.test('simple')
