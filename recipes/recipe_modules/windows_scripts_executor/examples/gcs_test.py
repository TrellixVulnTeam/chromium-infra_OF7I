# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from PB.recipes.infra.windows_image_builder import actions
from PB.recipes.infra.windows_image_builder import (offline_winpe_customization
                                                    as winpe)
from PB.recipes.infra.windows_image_builder import sources as sources
from PB.recipes.infra.windows_image_builder import (windows_image_builder as
                                                    wib)

from recipe_engine.post_process import DropExpectation, StatusFailure
from recipe_engine.post_process import StatusSuccess, StepCommandRE

DEPS = [
    'depot_tools/gsutil',
    'windows_scripts_executor',
    'recipe_engine/properties',
    'recipe_engine/platform',
    'recipe_engine/json',
]

PROPERTIES = wib.Image


def RunSteps(api, image):
  api.windows_scripts_executor.pin_wib_config(image)
  api.windows_scripts_executor.download_wib_artifacts(image)


def GenTests(api):
  # actions for adding files
  ACTION_ADD_PING = actions.Action(
      add_file=actions.AddFile(
          name='add ping from GCS',
          src=sources.Src(
              gcs_src=sources.GCSSrc(bucket='WinTools', source='net/ping.exe'),
          ),
          dst='Windows\\System32',
      ))

  yield (api.test('Add gcs src in action', api.platform('win', 64)) +
         api.properties(
             wib.Image(
                 name='win10Img',
                 arch=wib.ARCH_X86,
                 customizations=[
                     wib.Customization(
                         offline_winpe_customization=winpe
                         .OfflineWinPECustomization(
                             name='offWpeCust',
                             offline_customization=[
                                 actions.OfflineAction(
                                     name='action-1', actions=[ACTION_ADD_PING])
                             ]))
                 ])) + api.post_process(StatusSuccess) +  # recipe should pass
         api.post_process(DropExpectation))
