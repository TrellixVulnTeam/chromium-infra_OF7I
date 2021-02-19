# Copyright 2021 The LUCI Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""Test chrome-golo repo DHCP configs using dhcpd binaries via docker."""

import collections

from recipe_engine import post_process

DEPS = [
    'depot_tools/bot_update',
    'depot_tools/gclient',
    'depot_tools/tryserver',
    'infra/docker',
    'recipe_engine/buildbucket',
    'recipe_engine/cipd',
    'recipe_engine/file',
    'recipe_engine/json',
    'recipe_engine/path',
    'recipe_engine/platform',
    'recipe_engine/properties',
    'recipe_engine/raw_io',
    'recipe_engine/step',
]

_TEST_HOST = 'test-host'
_TEST_OS_VERSION = 'test_version'
_TEST_UFS_OUTPUT = [{
    'name': _TEST_HOST,
    'osVersion': {
        'value': 'Linux Ubuntu %s' % _TEST_OS_VERSION
    }
}]
_TEST_ZONE = 'test_zone'
_ZONE_HOST_MAP_FILE = 'services/dhcpd/zone_host_map.json'
_ZONE_HOST_MAP_TESTDATA = {_TEST_ZONE: [_TEST_HOST]}


def _GetDhcpOsVersions(api, zone_host_map):
  os_versions = {host: '' for hosts in zone_host_map.values() for host in hosts}

  packages_dir = api.path['cleanup'].join('packages')
  ensure_file = api.cipd.EnsureFile()
  ensure_file.add_package('infra/shivas/${platform}', 'prod')
  api.cipd.ensure(packages_dir, ensure_file)

  shivas = packages_dir.join('shivas')
  for host_type in ['host', 'vm']:
    step_result = api.step(
        'get ufs %s data' % host_type,
        [shivas, 'get', host_type, '-json', '-namespace', 'browser', '-noemit'],
        infra_step=True,
        stdout=api.json.output())
    for entry in step_result.stdout:
      name = entry['name']
      if name in os_versions:
        # chromeBrowserMachineLse key only exists for hosts but not vms.
        os = entry.get('chromeBrowserMachineLse', entry)['osVersion']['value']
        os_versions[name] = os.split(' ')[-1]
  return os_versions


def RunSteps(api):
  dhcp_map = collections.defaultdict(dict)
  images_needed = set()
  zones_to_test = set()

  assert api.platform.is_linux, 'Unsupported platform, only Linux is supported.'
  api.docker.ensure_installed()

  api.gclient.set_config('chrome_golo')
  api.bot_update.ensure_checkout()
  api.gclient.runhooks()

  api.docker.login(server='gcr.io', project='chromium-container-registry')

  patch_root = api.gclient.get_gerrit_patch_root()
  assert patch_root, ('local path is not configured for %s' %
                      api.m.tryserver.gerrit_change_repo_url)

  # Read a file in the repo that defines zones and their dhcp servers.
  zone_host_map = api.file.read_json(
      'read %s' % _ZONE_HOST_MAP_FILE,
      api.path['checkout'].join(_ZONE_HOST_MAP_FILE),
      test_data=_ZONE_HOST_MAP_TESTDATA)

  # Get up to date server OS versions from UFS.
  host_os_map = _GetDhcpOsVersions(api, zone_host_map)

  # Build a zone: os: hosts map for determining test that need to run.
  for zone, hosts in zone_host_map.iteritems():
    dhcp_map[zone] = {}
    for host in hosts:
      os_version = host_os_map[host]
      dhcp_map[zone].setdefault(os_version, []).append(host)

  # Iterate over files in the CL to determine which tests need to run.
  dhcp_dirs = ['%s/dhcpd' % d for d in ['configs', 'services']]
  for f in api.m.tryserver.get_files_affected_by_patch(patch_root):
    for zone in zone_host_map:
      if any(['%s/%s/' % (d, zone) in f for d in dhcp_dirs]):
        zones_to_test.add(zone)
        images_needed.update(dhcp_map[zone])

  # Pull docker images before testing with them.
  for os_version in sorted(images_needed):
    image = 'fleet_systems:dhcp_%s' % os_version
    try:
      api.docker.pull(image)
    except api.step.StepFailure:
      raise api.step.InfraFailure(
          'Image %s does not exist in the container registry.' % image)

  # Test each zone/os combination as necessary.
  for zone in sorted(zones_to_test):
    for os_version in sorted(dhcp_map[zone]):
      image = 'fleet_systems:dhcp_%s' % os_version
      api.docker.run(
          image='fleet_systems:dhcp_%s' % os_version,
          step_name='DHCP config test for %s on %s hosts: %s' %
          (zone, os_version, ', '.join(dhcp_map[zone][os_version])),
          cmd_args=[zone],
          dir_mapping=[(api.path['checkout'], '/src')])


def GenTests(api):
  stdout = api.json.output(_TEST_UFS_OUTPUT)

  def changed_files():
    test_file = 'services/dhcpd/%s/foo' % _TEST_ZONE
    t = api.override_step_data(
        'git diff to analyze patch', stdout=api.raw_io.output(test_file))
    t += api.path.exists(api.path['checkout'].join('chrome_golo', test_file))
    return t

  yield api.test(
      'chrome_golo_dhcp',
      api.properties(),
      api.buildbucket.try_build(),
      changed_files(),
      api.override_step_data('get ufs host data', stdout=stdout),
      api.override_step_data('get ufs vm data', stdout=stdout),
  )

  yield api.test(
      'chrome_golo_dhcp_image_does_not_exist',
      api.properties(),
      api.buildbucket.try_build(),
      changed_files(),
      api.override_step_data('get ufs host data', stdout=stdout),
      api.override_step_data('get ufs vm data', stdout=stdout),
      api.override_step_data(
          'docker pull fleet_systems:dhcp_%s' % _TEST_OS_VERSION, retcode=1),
      api.post_process(post_process.StatusException),
      api.post_process(post_process.DropExpectation),
  )
