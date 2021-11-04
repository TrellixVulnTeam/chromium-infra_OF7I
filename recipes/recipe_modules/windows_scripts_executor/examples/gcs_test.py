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
from RECIPE_MODULES.infra.windows_scripts_executor import test_helper as t

DEPS = [
    'depot_tools/gsutil',
    'windows_scripts_executor',
    'recipe_engine/properties',
    'recipe_engine/platform',
    'recipe_engine/json',
    'recipe_engine/raw_io',
    'recipe_engine/path',
]

PROPERTIES = wib.Image

image = 'gcs_test_image'
customization = 'gcs_customizations'
key = '9e007120d0ca02d6d82cf2bf23544ba222e9260eded07310393eca73a501842e'


def RunSteps(api, config):
  api.windows_scripts_executor.init(config)
  api.windows_scripts_executor.pin_available_sources()
  api.windows_scripts_executor.gen_canonical_configs(config)
  api.windows_scripts_executor.download_available_packages()
  api.windows_scripts_executor.execute_config(config)
  api.path.mock_add_paths('[CACHE]\\GCSPkgs\\chrome-gce-images\\' +
                          'WIB-WIM\\{}.wim'.format(key))
  api.windows_scripts_executor.upload_wib_artifacts()


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

  WPE_IMAGE_WITH_SRC = t.WPE_IMAGE(image, wib.ARCH_X86, customization,
                                   'no-action', [])
  tmp_customization = WPE_IMAGE_WITH_SRC.customizations[0]
  tmp_customization.offline_winpe_customization.image_src.CopyFrom(
      sources.Src(
          gcs_src=sources.GCSSrc(
              bucket='chrome-gce-images',
              source='WIB-OUT/intermediate-winpe.wim')))

  WPE_IMAGE_WITH_DEST = t.WPE_IMAGE(image, wib.ARCH_X86, customization,
                                    'no-action', [])
  tmp_customization = WPE_IMAGE_WITH_DEST.customizations[0]
  tmp_customization.offline_winpe_customization.image_dest.CopyFrom(
      sources.GCSSrc(
          bucket='chrome-gce-images', source='WIB-OUT/intermediate-winpe.wim'))

  # add unpinned artifact from gcs
  yield (
      api.test('Add unpinned binary from gcs', api.platform('win', 64)) +
      api.properties(
          t.WPE_IMAGE(image, wib.ARCH_X86, customization,
                      'add artifact from gcs', [ACTION_ADD_PING])) +
      # mock all the init and deint steps
      t.MOCK_WPE_INIT_DEINIT_SUCCESS(api, key, 'x86', image, customization) +
      # self pinned gcs artifact
      t.GCS_PIN_FILE(api, 'gs://WinTools/net/ping.exe') +
      # download the unpinned artifact
      t.GCS_DOWNLOAD_FILE(api, 'WinTools', 'net/ping.exe') +
      # add the given file to the image
      t.ADD_GCS_FILE(api, 'WinTools', 'net\\ping.exe', image, customization) +
      # assert that the generated image was uploaded
      t.CHECK_GCS_UPLOAD(
          api, '\[CACHE\]\\\\GCSPkgs\\\\chrome-gce-images' +
          '\\\\WIB-WIM\\\\{}.wim'.format(key),
          'gs://chrome-gce-images/WIB-WIM/{}.wim'.format(key)) +
      api.post_process(StatusSuccess) +  # recipe should pass
      api.post_process(DropExpectation))

  # add non-existent artifact from gcs
  yield (
      api.test('Add non-existent binary from gcs', api.platform('win', 64)) +
      api.properties(
          t.WPE_IMAGE(image, wib.ARCH_X86, customization,
                      'add artifact from gcs', [ACTION_ADD_PING])) +
      # mock all the init and deint steps
      t.MOCK_WPE_INIT_DEINIT_FAILURE(api, 'x86', image, customization) +
      # non-existent gcs artifact
      t.GCS_PIN_FILE(api, 'gs://WinTools/net/ping.exe', success=False) +
      # failure adding the file to the image
      t.ADD_GCS_FILE(
          api, 'WinTools', 'net\\ping.exe', image, customization, success=False)
      + api.post_process(StatusFailure) +  # recipe should fail
      api.post_process(DropExpectation))

  # Test using GCSSrc as an input image to the customization and input artifact
  # to add file action. Test both pinned (intermediate-winpe) and
  # unpinned(ping.exe) gcs srcs. Check for successful upload
  yield (
      api.test('Add customization src image', api.platform('win', 64)) +
      api.properties(WPE_IMAGE_WITH_SRC) +
      # mock all the init and deint steps
      t.MOCK_WPE_INIT_DEINIT_SUCCESS(api, key, 'x86', image, customization) +
      # Pin the given file to another gcs artifact
      t.GCS_PIN_FILE(
          api, 'gs://chrome-gce-images/WIB-OUT/' + 'intermediate-winpe.wim',
          'gs://chrome-gce-images/WIB-WIM/ffaa037563.wim') +
      # download the artifact from it's original link
      t.GCS_DOWNLOAD_FILE(api, 'chrome-gce-images', 'WIB-WIM/ffaa037563.wim') +
      # assert that the generated image was uploaded
      t.CHECK_GCS_UPLOAD(
          api, '\[CACHE\]\\\\GCSPkgs\\\\chrome-gce-images' +
          '\\\\WIB-WIM\\\\{}.wim'.format(key),
          'gs://chrome-gce-images/' + 'WIB-WIM/{}.wim'.format(key)) +
      api.post_process(StatusSuccess) +  # recipe should pass
      api.post_process(DropExpectation))

  # Test using GCSSrc as an output destination.
  yield (api.test('Add custom gcs destination', api.platform('win', 64)) +
         api.properties(WPE_IMAGE_WITH_DEST) +
         # mock all the init and deint steps
         t.MOCK_WPE_INIT_DEINIT_SUCCESS(api, key, 'x86', image, customization) +
         # assert that the generated image was uploaded
         t.CHECK_GCS_UPLOAD(
             api, '\[CACHE\]\\\\GCSPkgs\\\\chrome-gce-images' +
             '\\\\WIB-WIM\\\\{}.wim'.format(key),
             'gs://chrome-gce-images/' + 'WIB-WIM/{}.wim'.format(key)) +
         # assert that the generated image was uploaded
         t.CHECK_GCS_UPLOAD(
             api,
             '\[CACHE\]\\\\GCSPkgs\\\\chrome-gce-images' +
             '\\\\WIB-WIM\\\\{}.wim'.format(key),
             'gs://chrome-gce-images/WIB-OUT/' + 'intermediate-winpe.wim',
             orig='gs://chrome-gce-images/' + 'WIB-WIM/{}.wim'.format(key)) +
         api.post_process(StatusSuccess) +  # recipe should pass
         api.post_process(DropExpectation))
