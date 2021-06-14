# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from recipe_engine.post_process import DropExpectation
from PB.recipes.infra.windows_image_builder import windows_image_builder as wib

DEPS = [
    'windows_adk',
    'recipe_engine/properties',
]

PROPERTIES = wib.Image


def RunSteps(api, image):
  api.windows_adk.execute_config(image)


def GenTests(api):
  yield (api.test('basic') + api.properties(
      name='test',
      offline_winpe_customization=wib.OfflineCustomization(
          name='wpe test',
          offline_customization=[wib.OfflineAction(name='add,delete,rename')],
      )) + api.post_process(DropExpectation))
