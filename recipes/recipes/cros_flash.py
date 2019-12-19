# -*- coding: utf-8 -*-
# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""
This recipe is used to flash a CrOS DUT on a Chromium bot.

This essentially calls out to the cros-sdk's flash tool (located at
https://codesearch.chromium.org/chromium/src/third_party/chromite/cli/flash.py).

That tool however has a dependency on the cros SDK's chroot (see
https://chromium.googlesource.com/chromiumos/docs/+/master/developer_guide.md#Create-a-chroot
for more info). Though it's often used in CrOS development, the chroot isn't
found in Chromium development at all, and has very limited support on Chromium
bots. Consequently, this recipe will take care of setting it up prior to
flashing the DUT. The basic steps of this recipe are:

- Fetch a full ChromiumOS checkout via repo. The checkout will be placed in a
  named cache for subsequent re-use.
- Build the chroot.
- Enter the chroot and flash the device.
"""

from PB.recipes.infra import cros_flash as cros_flash_pb
from recipe_engine import post_process

DEPS = [
    'build/chromite',
    'build/chromium',
    'build/chromium_checkout',
    'build/repo',
    'depot_tools/gclient',
    'recipe_engine/context',
    'recipe_engine/path',
    'recipe_engine/platform',
    'recipe_engine/properties',
    'recipe_engine/python',
    'recipe_engine/raw_io',
    'recipe_engine/step',
]

# This is a special hostname that resolves to a different DUT depending on
# which swarming bot you're on.
CROS_DUT_HOSTNAME = 'variable_chromeos_device_hostname'

# Default password for root on a device running a test image. The contents of
# this password are public and not confidential.
CROS_SSH_PASSWORD = 'test0000'

# Path to an RSA key pair used for SSH auth with the DUT.
SWARMING_BOT_SSH_ID = '/b/id_rsa'

# Branch to sync the local ChromeOS checkout to.
# TODO(crbug.com/1025097): Pin this back on a release branch when one is cut
# that's stable.
CROS_BRANCH = 'master'

PROPERTIES = cros_flash_pb.Inputs


def RunSteps(api, properties):
  # After flashing, the host's ssh identity is no longer authorized with the
  # DUT, so we'll need to add it back. The host's identity is an ssh key file
  # located at SWARMING_BOT_SSH_ID that the swarming bot generates at start-up.
  # Ensure that file exists on the bot.
  api.path.mock_add_paths(SWARMING_BOT_SSH_ID)
  api.path.mock_add_paths(SWARMING_BOT_SSH_ID + '.pub')
  if (not api.path.exists(SWARMING_BOT_SSH_ID) or
      not api.path.exists(SWARMING_BOT_SSH_ID + '.pub')):
    api.python.failing_step('host ssh ID not found',  # pragma: no cover
        'The env var CROS_SSH_ID_FILE_PATH (%s) must be set and point to a ssh '
        'key pair to use for authentication with the DUT.' % (
            SWARMING_BOT_SSH_ID))

  # We really just need chromite, but it's easy to get a plain chromium
  # checkout via bot_update, and avoids us having to write our own `git clone`
  # and/or `git pull` logic.
  api.gclient.set_config('chromium')
  api.chromium.set_config('chromium')
  api.chromium_checkout.ensure_checkout({})

  # chromite's own virtual env setup conflicts with vpython, so temporarily
  # subvert vpython for the duration of the flash.
  src_dir = api.path['cache'].join('builder', 'src')
  with api.context(cwd=src_dir):
    with api.chromite.with_system_python():
      chromite_bin_path = src_dir.join('third_party', 'chromite', 'bin')
      arg_list = [
          'flash',
          CROS_DUT_HOSTNAME,
          properties.xbuddy_path,
          '--disable-rootfs-verification',  # Needed to add ssh ID below.
          '--clobber-stateful',  # Fully wipe the device.
          '--force',  # Force yes to all Y/N prompts.
          '--debug',  # More verbose logging.
      ]
      api.python('flash DUT', chromite_bin_path.join('cros'), arg_list)

  # Reauthorize the host's ssh identity with the DUT via ssh-copy-id, using
  # sshpass to pass in the root password.
  cmd = [
     '/usr/bin/sshpass',
     '-p', CROS_SSH_PASSWORD,
     '/usr/bin/ssh-copy-id', '-i', SWARMING_BOT_SSH_ID + '.pub',
     'root@' + CROS_DUT_HOSTNAME,
  ]
  api.step('reauthorize DUT ssh access', cmd)


def GenTests(api):
  yield api.test(
      'basic_test',
      api.platform('linux', 64),
      api.properties(xbuddy_path='xbuddy://some/image/path',),
      api.post_process(post_process.StatusSuccess),
      api.post_process(post_process.DropExpectation)
  )
