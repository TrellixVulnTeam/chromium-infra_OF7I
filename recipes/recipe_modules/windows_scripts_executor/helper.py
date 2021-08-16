# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from PB.recipes.infra.windows_image_builder import windows_image_builder as wib


def conv_to_win_path(path):
  """ Converts unix paths to windows ones."""
  return '\\'.join(path.split('/'))


def iter_src(config, oper):
  """ iter_src iterates through all the src configs and runs oper on them."""
  wpec = config.offline_winpe_customization
  if wpec:
    for off_action in wpec.offline_customization:
      for action in off_action.actions:
        #TODO(anushruth): Update to include other actions
        if action.WhichOneof('action') == 'add_file':
          oper(action.add_file.src)
