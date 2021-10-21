# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from PB.recipes.infra.windows_image_builder import windows_image_builder as wib
from PB.recipes.infra.windows_image_builder import actions

# Format strings for use in mount cmdline options
EDIT_OFFLINE_REG_CMD = 'Edit-OfflineRegistry'
EDIT_OFFLINE_REG_IMG_PATH = '-OfflineImagePath "{}"'
EDIT_OFFLINE_REG_HIVE_FILE = '-OfflineRegHiveFile "{}"'
EDIT_OFFLINE_REG_KEY_PATH = '-RegistryKeyPath "{}"'
EDIT_OFFLINE_REG_PROPERTY_NAME = '-PropertyName "{}"'
EDIT_OFFLINE_REG_PROPERTY_VALUE = '-PropertyValue "{}"'
EDIT_OFFLINE_REG_PROPERTY_TYPE = '-PropertyType "{}"'


def edit_offline_registry(powershell, res, edit_offline_registry_action, img):
  action = edit_offline_registry_action

  ptype = actions.RegPropertyType.Name(action.property_type).upper()

  args = [
      EDIT_OFFLINE_REG_IMG_PATH.format(img),
      EDIT_OFFLINE_REG_HIVE_FILE.format(action.reg_hive_file),
      EDIT_OFFLINE_REG_KEY_PATH.format(action.reg_key_path),
      EDIT_OFFLINE_REG_PROPERTY_NAME.format(action.property_name),
      EDIT_OFFLINE_REG_PROPERTY_VALUE.format(action.property_value),
      EDIT_OFFLINE_REG_PROPERTY_TYPE.format(ptype)
  ]

  reg_key_leaf = action.reg_key_path.split('\\')[-1]
  name = 'Edit Offline Registry Key {} and Property {}'.format(
      reg_key_leaf, action.property_name)

  powershell(name, res.join(EDIT_OFFLINE_REG_CMD), args=args)
