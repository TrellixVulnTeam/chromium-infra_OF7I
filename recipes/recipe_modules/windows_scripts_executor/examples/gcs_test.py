# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from PB.recipes.infra.windows_image_builder import actions
from PB.recipes.infra.windows_image_builder import (offline_winpe_customization
                                                    as winpe)
from PB.recipes.infra.windows_image_builder import sources as sources
from PB.recipes.infra.windows_image_builder import dest as dest
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
  api.path.mock_add_paths('[CACHE]\\Pkgs\\GCSPkgs\\WinTools\\net\\ping.exe',
                          'FILE')
  # mock existence of the image pulled from GCS
  api.path.mock_add_paths(
      '[CACHE]\\Pkgs\\GCSPkgs\\chrome-gce-images\\WIB-WIM\\ffaa037563.wim',
      'FILE')
  api.windows_scripts_executor.execute_config(config)
  # mock existence of image created using copype
  api.path.mock_add_paths('[CLEANUP]\\{}\\workdir\\'.format(customization) +
                          'media\\sources\\boot.wim')
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
  tmp_customization.offline_winpe_customization.image_dests.append(
      dest.Dest(
          gcs_src=sources.GCSSrc(
              bucket='chrome-gce-images',
              source='WIB-OUT/intermediate-winpe.wim')))

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
          api, '\[CLEANUP\]\\\\{}\\\\workdir\\\\media'.format(customization) +
          '\\\\sources\\\\boot.wim',
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
      t.MOCK_WPE_INIT_DEINIT_FAILURE(api, key, 'x86', image, customization) +
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
      # mock check for customization output existence
      t.MOCK_CUST_OUTPUT(
          api, image, 'gs://chrome-gce-images/WIB-WIM/{}.wim'.format(key),
          False) + t.MOUNT_WIM(api, 'x86', image, customization) +
      t.UMOUNT_WIM(api, image, customization) +
      t.DEINIT_WIM_ADD_CFG_TO_ROOT(api, key, image, customization) +
      t.CHECK_UMOUNT_WIM(api, image, customization) +
      # Pin the given file to another gcs artifact
      t.GCS_PIN_FILE(api,
                     'gs://chrome-gce-images/WIB-OUT/intermediate-winpe.wim',
                     'gs://chrome-gce-images/WIB-WIM/ffaa037563.wim') +
      # download the artifact from it's original link
      t.GCS_DOWNLOAD_FILE(api, 'chrome-gce-images', 'WIB-WIM/ffaa037563.wim') +
      # assert that the generated image was uploaded
      t.CHECK_GCS_UPLOAD(
          api, '\[CACHE\]\\\\Pkgs\\\\GCSPkgs\\\\chrome-gce-images\\\\' +
          'WIB-WIM\\\\ffaa037563.wim',
          'gs://chrome-gce-images/WIB-WIM/{}.wim'.format(key)) +
      api.post_process(StatusSuccess) +  # recipe should pass
      api.post_process(DropExpectation))

  # Test using GCSSrc as an output destination.
  yield (
      api.test('Add custom gcs destination', api.platform('win', 64)) +
      api.properties(WPE_IMAGE_WITH_DEST) +
      # mock all the init and deint steps
      t.MOCK_WPE_INIT_DEINIT_SUCCESS(api, key, 'x86', image, customization) +
      # assert that the generated image was uploaded
      t.CHECK_GCS_UPLOAD(
          api, '\[CLEANUP\]\\\\{}\\\\workdir\\\\media'.format(customization) +
          '\\\\sources\\\\boot.wim',
          'gs://chrome-gce-images/WIB-WIM/{}.wim'.format(key)) +
      # assert that the generated image was uploaded
      t.CHECK_GCS_UPLOAD(
          api,
          '\[CLEANUP\]\\\\{}\\\\workdir\\\\media'.format(customization) +
          '\\\\sources\\\\boot.wim',
          'gs://chrome-gce-images/WIB-OUT/intermediate-winpe.wim',
          orig='gs://chrome-gce-images/WIB-WIM/{}.wim'.format(key)) +
      api.post_process(StatusSuccess) +  # recipe should pass
      api.post_process(DropExpectation))
