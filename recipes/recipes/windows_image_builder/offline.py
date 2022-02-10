# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from recipe_engine import post_process
from google.protobuf.struct_pb2 import Struct

from PB.recipes.infra.windows_image_builder import input as input_pb
from PB.recipes.infra.windows_image_builder \
    import windows_image_builder as wib
from PB.recipes.infra.windows_image_builder \
    import offline_winpe_customization as owc
from PB.go.chromium.org.luci.buildbucket.proto \
  import builds_service as bs_pb2
from PB.go.chromium.org.luci.buildbucket.proto \
  import build as b_pb2
from PB.go.chromium.org.luci.buildbucket.proto \
  import builder as builder_pb2

from RECIPE_MODULES.infra.windows_scripts_executor \
    import test_helper as t

DEPS = [
    'depot_tools/bot_update',
    'depot_tools/gclient',
    'recipe_engine/context',
    'recipe_engine/file',
    'recipe_engine/json',
    'recipe_engine/path',
    'recipe_engine/platform',
    'recipe_engine/properties',
    'recipe_engine/proto',
    'recipe_engine/step',
    'recipe_engine/buildbucket',
    'recipe_engine/raw_io',
    'recipe_engine/runtime',
    'windows_adk',
    'windows_scripts_executor',
]

PYTHON_VERSION_COMPATIBILITY = 'PY3'

PROPERTIES = input_pb.Inputs

def RunSteps(api, inputs):
  """This recipe runs windows offline builder for a given user config."""
  if not api.platform.is_win:
    raise AssertionError("This recipe only runs on Windows")

  if not inputs.config_path:
    raise api.step.StepFailure("`config_path` is a required property")

  builder_named_cache = api.path['cache'].join('builder')

  configs = []

  with api.step.nest('read user config') as c:
    # download the configs repo
    api.gclient.set_config('infradata_config')
    api.gclient.c.solutions[0].revision = 'origin/main'
    with api.context(cwd=builder_named_cache):
      api.bot_update.ensure_checkout()
      api.gclient.runhooks()
      # split the string on '/' as luci scheduler passes a unix path and this
      # recipe is expected to run on windows ('\')
      cfg_path = builder_named_cache.join('infra-data-config',
                                          *inputs.config_path.split('/'))

      # Recursively call the offline.py recipe with all configs
      cfgs = api.file.listdir(
          "Read all the configs",
          cfg_path,
          test_data=['first.cfg', 'second.cfg'])
      reqs = []
      for cfg in cfgs:
        if str(cfg).endswith('.cfg'):
          try:
            configs.append(
                api.file.read_proto(
                    name='Reading ' + inputs.config_path,
                    source=cfg,
                    msg_class=wib.Image,
                    codec='TEXTPB',
                    test_proto=t.WPE_IMAGE(
                        image='test',
                        arch=wib.ARCH_X86,
                        customization='test_cust',
                        sub_customization='tests',
                        action_list=[])))
          except ValueError as e:  #pragma: no cover
            _, name = api.path.split(cfg)
            summary = c.step_summary_text
            summary += 'Failed to read {}: {} <br>'.format(name, e)
            c.step_summary_text = summary

  if not configs:
    # If there are no config files, exit
    return  #pragma: no cover

  # initialize the recipe_module
  api.windows_scripts_executor.init()

  # collect all the customizations from all the configs
  custs = []
  for config in configs:
    custs.extend(api.windows_scripts_executor.init_customizations(config))

  custs = api.windows_scripts_executor.process_customizations(custs)

  with api.step.nest('Execute customizations') as e:
    # check for any customizations that need execution
    exec_customizations = []
    if custs:
      exec_custs = api.windows_scripts_executor.get_executable_configs(custs)
      if exec_custs:
        for a in exec_custs:
          exec_customizations.append(a)

    # execute the customizations that need to be executed
    reqs = []
    for cust in exec_customizations:
      img = api.json.loads(api.proto.encode(cust, 'JSONPB'))
      reqs.append(
          api.buildbucket.schedule_request(
              builder='WinPE Customization Builder',
              properties=img,
          ))

    # TODO(anushruth): Avoid executing duplicate customizations based on key
    if reqs:

      def url_title(build):
        """ url_title is a helper function to display the customization
            name over the build link in schedule process.
            Returns string formatted with builder name and customization
        """
        props = build.input.properties
        return '[{}] {}:{}'.format(
            build.builder.builder, props['name'],
            props['customizations'][0]['offline_winpe_customization']['name'])

      # schedule all the builds
      api.buildbucket.schedule(reqs, url_title_fn=url_title)
    else:
      e.step_summary_text = 'No customizations were executed'


def GenTests(api):

  key = '0ba325f4cf5356b9864719365a807f2c9d48bf882d333149cebd9d1ec0b64e7b'
  image = 'test'
  cust = 'test_cust'

  # Mock schedule requests batch response
  prop = b_pb2.Build.Input()
  prop.properties['name'] = image
  prop.properties['customizations'] = [{
      'offline_winpe_customization': {
          'name': cust
      }
  }]
  BATCH_RESPONSE = bs_pb2.BatchResponse(responses=[
      dict(
          schedule_build=dict(
              builder=dict(builder='WinPE Customization Builder'), input=prop)),
  ])

  # Test builds scheduled case
  yield (api.test('basic_scheduled', api.platform('win', 64)) +
         api.properties(input_pb.Inputs(config_path="test_config")) +
         t.MOCK_CUST_OUTPUT(
             api, 'gs://chrome-gce-images/WIB-WIM/{}.zip'.format(key), False) +
         # mock schedule output to test builds scheduled state
         api.buildbucket.simulated_schedule_output(
             BATCH_RESPONSE,
             step_name='Execute customizations.buildbucket.schedule') +
         api.post_process(post_process.StatusSuccess))

  # Test builds not scheduled case
  yield (api.test('basic_no_scheduled', api.platform('win', 64)) +
         api.properties(input_pb.Inputs(config_path="test_config")) +
         t.MOCK_CUST_OUTPUT(
             api, 'gs://chrome-gce-images/WIB-WIM/{}.zip'.format(key), True) +
         api.post_process(post_process.StatusSuccess))

  yield (
      api.test('not_run_on_windows', api.platform('linux', 64)) +
      api.properties(
          input_pb.Inputs(
              config_path="test_config",
          ),
      ) +
      api.expect_exception('AssertionError'))

  yield (api.test('run_without_config_path', api.platform('win', 64)) +
         api.properties(input_pb.Inputs(config_path="",),) +
         api.post_process(post_process.StatusFailure))

  #yield (api.test('run_with_multiple_configs', api.platform('win', 64)) +
  #       api.properties(input_pb.Inputs(config_path="test_path"))+
