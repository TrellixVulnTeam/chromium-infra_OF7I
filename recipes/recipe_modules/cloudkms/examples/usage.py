# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

DEPS = [
  'cloudkms',
  'recipe_engine/path',
]


def RunSteps(api):
  api.cloudkms.decrypt(
    'projects/PROJECT/locations/global/keyRings/KEYRING/cryptoKeys/KEY',
    api.path['start_dir'].join('ciphertext'),
    api.path['cleanup'].join('plaintext'),
  )
  # Decrypt another file; the module shouldn't install cloudkms again.
  api.cloudkms.decrypt(
    'projects/PROJECT/locations/global/keyRings/KEYRING/cryptoKeys/KEY',
    api.path['start_dir'].join('encrypted'),
    api.path['cleanup'].join('decrypted'),
  )

  api.cloudkms.sign(
      'projects/PROJECT/locations/LOCATION/keyRings/KEYRING/cryptoKeys/KEY',
      api.path['start_dir'].join('chrome_build'),
      api.path['start_dir'].join('signed_bin'),
  )
  #Sign another file; with service_account_json file not None
  api.cloudkms.sign(
      'projects/PROJECT/locations/LOCATION/keyRings/KEYRING/cryptoKeys/KEY',
      api.path['start_dir'].join('build'),
      api.path['start_dir'].join('bin'),
      'service_acc'
  )

  api.cloudkms.verify(
      'projects/PROJECT/locations/LOCATION/keyRings/KEYRING/cryptoKeys/KEY',
      api.path['start_dir'].join('signed_chrome'),
      api.path['start_dir'].join('signature'),
      api.path['cleanup'].join('result'),
  )
  #Sign another file; with service_account_json file not None
  api.cloudkms.verify(
      'projects/PROJECT/locations/LOCATION/keyRings/KEYRING/cryptoKeys/KEY',
      api.path['start_dir'].join('signed'),
      api.path['start_dir'].join('sign'),
      api.path['cleanup'].join('status'),
      'service_acc'
  )


def GenTests(api):
  yield api.test('simple')
