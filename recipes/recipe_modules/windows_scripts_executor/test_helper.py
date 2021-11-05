# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from PB.recipes.infra.windows_image_builder import (offline_winpe_customization
                                                    as winpe)
from PB.recipes.infra.windows_image_builder import actions
from PB.recipes.infra.windows_image_builder import windows_image_builder as wib
from PB.recipes.infra.windows_image_builder import sources

from recipe_engine.post_process import DropExpectation, StatusFailure
from recipe_engine.post_process import StatusSuccess, StepCommandRE

#    Step data mock methods. Use these to mock the step outputs

# _gcs_stat is the mock output of gsutil stat command
_gcs_stat = """
{}:
    Creation time:          Tue, 12 Oct 2021 00:32:06 GMT
    Update time:            Tue, 12 Oct 2021 00:32:06 GMT
    Storage class:          STANDARD
    Content-Length:         658955236
    Content-Type:           application/octet-stream
    Metadata:
        orig:               {}
    Hash (crc32c):          oaYUgQ==
    Hash (md5):             +W9+CqZbFtYTZrUrDPltMw==
    ETag:                   CJOHnM3Pw/MCEAE=
    Generation:             1633998726431635
    Metageneration:         1
"""


def NEST(*args):
  """ NEST generates nested names for steps """
  return '.'.join(args)


def NEST_CONFIG_STEP(image):
  """ generate config step name for nesting """
  return 'execute config {}'.format(image)


def NEST_WINPE_CUSTOMIZATION_STEP(customization):
  """ generate winpe customization step name for nesting """
  return 'offline winpe customization {}'.format(customization)


def NEST_WINPE_INIT_STEP(arch, customization):
  """ generate winpe init step nesting names """
  return 'Init WinPE image modification {}'.format(
      arch) + ' in [CLEANUP]\\{}\\workdir'.format(customization)


def NEST_WINPE_DEINIT_STEP():
  """ generate winpe deinit step nesting names """
  return 'Deinit WinPE image modification'


def NEST_PIN_ALL_SRCS():
  """ generate Pin Src step nesting name """
  return 'Pin all the required artifacts'


def NEST_DOWNLOAD_ALL_SRC():
  """ Download all available packages step name"""
  return 'Download all available packages'


def NEST_UPLOAD_ALL_SRC():
  """ Upload all gcs artifacts step name"""
  return 'Upload all pending gcs artifacts'


def json_res(api, success=True, err_msg='Failed step'):
  """ generate a api.json object to moxk outputs """
  if success:
    return api.json.output({'results': {'Success': success,}})
  return api.json.output({
      'results': {
          'Success': success,
          'Command': 'powershell',
          'ErrorInfo': {
              'Message': err_msg,
          },
      }
  })


def MOCK_WPE_INIT_DEINIT_SUCCESS(api, key, arch, image, customization):
  """ mock all the winpe init and deinit steps """
  return GEN_WPE_MEDIA(api, arch, image, customization) + MOUNT_WIM(
      api, arch, image, customization) + UMOUNT_WIM(
          api, image, customization) + DEINIT_WIM_ADD_CFG_TO_ROOT(
              api, key, image, customization) + CHECK_UMOUNT_WIM(
                  api, image, customization)


def MOCK_WPE_INIT_DEINIT_FAILURE(api, arch, image, customization):
  """ mock all the winpe init and deinit steps on an action failure """
  return GEN_WPE_MEDIA(api, arch, image, customization) + MOUNT_WIM(
      api, arch, image, customization) + UMOUNT_WIM(
          api, image, customization) + CHECK_UMOUNT_WIM(
              api, image, customization, save=False)


def GEN_WPE_MEDIA(api, arch, image, customization, success=True):
  """ Mock winpe media generation step """
  return api.step_data(
      NEST(
          NEST_CONFIG_STEP(image), NEST_WINPE_CUSTOMIZATION_STEP(customization),
          NEST_WINPE_INIT_STEP(arch, customization),
          'PowerShell> Gen WinPE media for {}'.format(arch)),
      stdout=json_res(api, success))


def MOUNT_WIM(api, arch, image, customization, success=True):
  """ mock mount winpe wim step """
  return api.step_data(
      NEST(
          NEST_CONFIG_STEP(image), NEST_WINPE_CUSTOMIZATION_STEP(customization),
          NEST_WINPE_INIT_STEP(arch, customization),
          'PowerShell> Mount wim to [CLEANUP]\\{}\\workdir\\mount'.format(
              customization)),
      stdout=json_res(api, success))


def UMOUNT_WIM(api, image, customization, success=True):
  """ mock unmount winpe wim step """
  return api.step_data(
      NEST(
          NEST_CONFIG_STEP(image), NEST_WINPE_CUSTOMIZATION_STEP(customization),
          NEST_WINPE_DEINIT_STEP(),
          'PowerShell> Unmount wim at [CLEANUP]\\{}\\workdir\\mount'.format(
              customization)),
      stdout=json_res(api, success))


def DEINIT_WIM_ADD_CFG_TO_ROOT(api, key, image, customization, success=True):
  """ mock add cfg to root step in wpe deinit """
  return api.step_data(
      NEST(
          NEST_CONFIG_STEP(image), NEST_WINPE_CUSTOMIZATION_STEP(customization),
          NEST_WINPE_DEINIT_STEP(),
          'PowerShell> Add cfg [CLEANUP]\\configs\\{}.cfg'.format(key)),
      stdout=json_res(api, success))


def GIT_PIN_FILE(api, refs, path, data):
  """ mock git pin file step """
  return api.step_data(
      NEST(
          NEST_PIN_ALL_SRCS(),
          'gitiles log: ' + '{}/{}'.format(refs, path),
      ),
      api.gitiles.make_log_test_data(data),
  )


def GCS_PIN_FILE(api, url, pin_url='', success=True):
  """ mock gcs pin file action"""
  retcode = 1
  if success:
    retcode = 0
  if not pin_url:
    pin_url = url
  return api.step_data(
      NEST(NEST_PIN_ALL_SRCS(), 'gsutil stat {}'.format(url)),
      api.raw_io.stream_output(_gcs_stat.format(url, pin_url)),
      retcode=retcode,
  )


def GCS_DOWNLOAD_FILE(api, bucket, source, success=True):
  """ mock gcs download file action"""
  retcode = 1
  if success:
    retcode = 0
  return api.step_data(
      NEST(NEST_DOWNLOAD_ALL_SRC(),
           'gsutil download gs://{}/{}'.format(bucket, source)),
      retcode=retcode,
  )


def GIT_FETCH_FILE(api, commit, path, data):
  """ mock git fetch step """
  return api.step_data(
      NEST(
          NEST_DOWNLOAD_ALL_SRC(),
          'fetch ' + '{}:{}'.format(commit, path),
      ), api.gitiles.make_encoded_file(data))


def ADD_GIT_FILE(api, image, customization, commit, path, success=True):
  """ mock add git file to unpacked image step """
  return ADD_FILE(api, image, customization,
                  '[CACHE]\\Pkgs\\GITPkgs\\' + commit + '\\' + path, success)


def ADD_FILE(api, image, customization, path, success=True):
  """ mock add file to image step """
  return api.step_data(
      NEST(
          NEST_CONFIG_STEP(image), NEST_WINPE_CUSTOMIZATION_STEP(customization),
          'PowerShell> Add file {}'.format(path)),
      stdout=json_res(api, success))


def ADD_CIPD_FILE(api, pkg, platform, image, customization, success=True):
  """ mock add cipd file to unpacked image step """
  return ADD_FILE(
      api, image, customization,
      '[CACHE]\\Pkgs\\CIPDPkgs\\resolved-instance_id-of-latest----------' +
      '\\{}\\{}'.format(pkg, platform), success)


def ADD_GCS_FILE(api, bucket, path, image, customization, success=True):
  """ mock add cipd file to unpacked image step """
  return ADD_FILE(api, image, customization,
                  '[CACHE]\\Pkgs\\GCSPkgs\\{}\\{}'.format(bucket,
                                                          path), success)


def INSTALL_FILE(api, name, image, customization, success=True):
  """ mock install file to image step """
  return api.step_data(
      NEST(
          NEST_CONFIG_STEP(image), NEST_WINPE_CUSTOMIZATION_STEP(customization),
          'PowerShell> Install package {}'.format(name)),
      stdout=json_res(api, success))


def EDIT_REGISTRY(api, name, image, customization, success=True):
  """ mock registry edit action step"""
  return api.step_data(
      NEST(
          NEST_CONFIG_STEP(image), NEST_WINPE_CUSTOMIZATION_STEP(customization),
          'PowerShell> Edit Offline Registry Key Features and Property {}'
          .format(name)),
      stdout=json_res(api, success))



#    Assert methods to validate that a certain step was run


def CHECK_UMOUNT_WIM(api, image, customization, save=True):
  """
      Post check that the wim was unmounted with either save or discard
  """
  args = ['.*'] * 11  # ignore matching the first 11 terms
  # check the last option
  if not save:
    args.append('-Discard')
  else:
    args.append('-Save')
  return api.post_process(
      StepCommandRE,
      NEST(
          NEST_CONFIG_STEP(image), NEST_WINPE_CUSTOMIZATION_STEP(customization),
          NEST_WINPE_DEINIT_STEP(),
          'PowerShell> Unmount wim at [CLEANUP]\\{}\\workdir\\mount'.format(
              customization)), args)


def CHECK_GCS_UPLOAD(api, source, destination, orig=''):
  """
      Post check the upload to GCS
  """
  if not orig:
    orig = destination
  args = ['.*'] * 11
  args[7] = 'x-goog-meta-orig:{}'.format(orig)  # ensure the orig meta url
  args[9] = source  # ensure the correct local src
  args[10] = destination  # ensure upload to correct location
  return api.post_process(
      StepCommandRE,
      NEST(NEST_UPLOAD_ALL_SRC(), 'gsutil upload {}'.format(destination)), args)


def CHECK_INSTALL_CAB(api, image, customization, action, args=None):
  """
      Post check for installation
  """
  wild_card = ['.*'] * 12
  if args:
    wild_card.append(*args)
  return api.post_process(
      StepCommandRE,
      NEST(
          NEST_CONFIG_STEP(image), NEST_WINPE_CUSTOMIZATION_STEP(customization),
          'PowerShell> Install package {}'.format(action)), wild_card)

#   Generate proto configs helper functions


def WPE_IMAGE(image, arch, customization, sub_customization, action_list):
  """ generates a winpe customization image """
  return wib.Image(
      name=image,
      arch=arch,
      customizations=[
          wib.Customization(
              offline_winpe_customization=winpe.OfflineWinPECustomization(
                  name=customization,
                  offline_customization=[
                      actions.OfflineAction(
                          name=sub_customization, actions=action_list)
                  ]))
      ])
