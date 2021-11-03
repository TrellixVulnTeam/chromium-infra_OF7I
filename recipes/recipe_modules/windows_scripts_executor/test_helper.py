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


def NEST(*args):
  """ NEST generates nested names for steps """
  return '.'.join(args)


def NEST_CONFIG_STEP(image):
  """ generate config step name for nesting """
  return 'execute config {}'.format(image)


def NEST_WINPE_CUSTOMIZATION_STEP(customization):
  """ generate winpe customization step name for nesting """
  return 'offline winpe customization {}'.format(customization)


def NEST_WINPE_INIT_STEP(arch):
  """ generate winpe init step nesting names """
  return 'Init WinPE image modification {} in [CACHE]\\WinPEImage'.format(arch)


def NEST_WINPE_DEINIT_STEP():
  """ generate winpe deinit step nesting names """
  return 'Deinit WinPE image modification'


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
          NEST_WINPE_INIT_STEP(arch),
          'PowerShell> Gen WinPE media for {}'.format(arch)),
      stdout=json_res(api, success))


def MOUNT_WIM(api, arch, image, customization, success=True):
  """ mock mount winpe wim step """
  return api.step_data(
      NEST(
          NEST_CONFIG_STEP(image), NEST_WINPE_CUSTOMIZATION_STEP(customization),
          NEST_WINPE_INIT_STEP(arch),
          'PowerShell> Mount wim to [CACHE]\\WinPEImage\\mount'),
      stdout=json_res(api, success))


def UMOUNT_WIM(api, image, customization, success=True):
  """ mock unmount winpe wim step """
  return api.step_data(
      NEST(
          NEST_CONFIG_STEP(image), NEST_WINPE_CUSTOMIZATION_STEP(customization),
          NEST_WINPE_DEINIT_STEP(),
          'PowerShell> Unmount wim at [CACHE]\\WinPEImage\\mount'),
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
      'Pin git artifacts to refs.gitiles log: ' + '{}/{}'.format(refs, path),
      api.gitiles.make_log_test_data(data),
  )


def GIT_FETCH_FILE(api, commit, path, data):
  """ mock git fetch step """
  return api.step_data(
      'Get all git artifacts.fetch ' + '{}:{}'.format(commit, path),
      api.gitiles.make_encoded_file(data))


def ADD_GIT_FILE(api, image, customization, commit, path, success=True):
  """ mock add git file to unpacked image step """
  return ADD_FILE(api, image, customization,
                  '[CACHE]\\GITPkgs\\' + commit + '\\' + path, success)


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
      '[CACHE]\\' + 'CIPDPkgs\\resolved-instance_id-of-latest----------' +
      '\\{}\\{}'.format(pkg, platform), success)


def INSTALL_FILE(api, name, image, customization, success=True):
  """ mock install file to image step """
  return api.step_data(
      NEST(
          NEST_CONFIG_STEP(image), NEST_WINPE_CUSTOMIZATION_STEP(customization),
          'PowerShell> Install package {}'.format(name)),
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
          'PowerShell> Unmount wim at [CACHE]\\WinPEImage\\mount'), args)


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
