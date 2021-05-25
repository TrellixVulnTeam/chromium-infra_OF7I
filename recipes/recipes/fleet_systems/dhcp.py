# Copyright 2021 The LUCI Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""Test chrome-golo repo DHCP configs using dhcpd binaries via docker."""

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
    'recipe_engine/python',
    'recipe_engine/raw_io',
    'recipe_engine/step',
]

_IMAGE_TEMPLATE = 'fleet_systems/dhcp/%s:latest'

# These image versions align with the docker image repository here:
# https://console.cloud.google.com/gcr/images/chops-public-images-prod/GLOBAL/fleet_systems/dhcp
#
# The images are generated from configs here:
# https://chromium.googlesource.com/infra/infra/+/master/build/images/daily/fleet_systems/dhcp
#
# UFS reports these versions for the DHCP servers (Ubuntu LTS only).
_IMAGE_VERSIONS = [14.04, 16.04, 18.04, 20.04]

_TEST_HOST = 'test-host'
_TEST_ZONE = 'test_zone'
_ZONE_HOST_MAP_FILE = 'services/dhcpd/zone_host_map.json'
_ZONE_HOST_MAP_TESTDATA = {_TEST_ZONE: [_TEST_HOST]}


def _GetZonesToTest(api, zone_host_map):
  """Iterate over files in the CL to determine which zones need testing."""
  dhcp_dirs = ['%s/dhcpd' % d for d in ['configs', 'services']]
  patch_root = api.gclient.get_gerrit_patch_root()
  zones_to_test = set()

  assert patch_root, ('local path is not configured for %s' %
                      api.m.tryserver.gerrit_change_repo_url)
  for f in api.m.tryserver.get_files_affected_by_patch(patch_root):
    for zone in zone_host_map:
      if any(['%s/%s/' % (d, zone) in f for d in dhcp_dirs]):
        zones_to_test.add(zone)
  return zones_to_test


def _GetShivasCIPD(api):
  """Install shivas CIPD package and return path to the binary."""
  packages_dir = api.path['cleanup'].join('packages')
  ensure_file = api.cipd.EnsureFile()
  ensure_file.add_package('infra/shivas/${platform}', 'prod')
  api.cipd.ensure(packages_dir, ensure_file)
  return packages_dir.join('shivas')


def _GetUFSData(api, names, shivas, sub_command):
  """Use Shivas to query UFS for host or vm information."""
  return api.step(
      'get ufs %s data' % sub_command,
      [shivas, 'get', sub_command, '-json', '-namespace', 'browser', '-noemit'
      ] + list(names),
      infra_step=True,
      stdout=api.json.output()).stdout


def _GetHostOsVersions(api, hosts):
  """Get up to date OS versions for each host from UFS."""
  host_os_map = {}
  shivas = _GetShivasCIPD(api)

  for sub_command in ['host', 'vm']:
    step_result_data = _GetUFSData(api, hosts, shivas, sub_command)
    for entry in step_result_data:
      host = entry['name']
      # chromeBrowserMachineLse key only exists for hosts but not vms.
      os = entry.get('chromeBrowserMachineLse', entry)['osVersion']['value']
      try:
        version = float(os.split('Linux Ubuntu ')[-1])  # 18.04, 20.04 etc.
      except ValueError:
        # Set a default value if something goes wrong.
        version = _IMAGE_VERSIONS[-1]

      # Find the closest image version to use: 12.04 would be 14.04,
      # 20.04 would be 20.04, 22.04 would be 20.04.
      closest_version = min(_IMAGE_VERSIONS, key=lambda x: abs(x - version))
      host_os_map[host] = closest_version
      if closest_version != version:
        api.python.succeeding_step(
            'WARNING %s: ' % host,
            'using \'Ubuntu %s\', no compatible image version \'%s\'' %
            (closest_version, os))
  missing_hosts = set(hosts).difference(host_os_map)
  if missing_hosts:
    api.python.succeeding_step(
        'WARNING UFS missing hosts:',
        '"%s" listed in "zone_host_map.json" but not present in UFS' %
        ', '.join(missing_hosts))
  return host_os_map


def _PullDockerImages(api, os_versions):
  """Pull docker images needed for testing."""
  api.docker.login(server='gcr.io', project='chops-public-images-prod')

  for os_version in sorted(os_versions):
    image = _IMAGE_TEMPLATE % os_version
    try:
      api.docker.pull(image)
    except api.step.StepFailure:
      raise api.step.InfraFailure(
          'Image %s does not exist in the container registry.' % image)


def RunSteps(api):
  hosts_to_test = set()
  os_versions_by_zone = {}

  assert api.platform.is_linux, 'Unsupported platform, only Linux is supported.'
  api.docker.ensure_installed()

  api.gclient.set_config('chrome_golo')
  api.bot_update.ensure_checkout()
  api.gclient.runhooks()

  # Read a file in the repo that defines zones and their dhcp servers.
  zone_host_map = api.file.read_json(
      'read %s' % _ZONE_HOST_MAP_FILE,
      api.path['checkout'].join(_ZONE_HOST_MAP_FILE),
      test_data=_ZONE_HOST_MAP_TESTDATA)

  zones_to_test = _GetZonesToTest(api, zone_host_map)
  if not zones_to_test:
    api.python.succeeding_step('CL does not contain DHCP changes', '')
    return

  # Determine hosts to test based on zones that have changes.
  for zone in zones_to_test:
    hosts_to_test.update(zone_host_map[zone])

  host_os_map = _GetHostOsVersions(api, hosts_to_test)

  _PullDockerImages(api, set(host_os_map.values()))

  for zone, hosts in zone_host_map.items():
    # Get list of os versions in this zone, skipping hosts not in UFS output.
    os_versions_by_zone[zone] = set(
        [host_os_map[host] for host in hosts if host in host_os_map])

  # Test each zone/os combination as necessary.
  for zone in sorted(zones_to_test):
    for os_version in os_versions_by_zone[zone]:
      # Get list of hosts in this zone at this os version.
      hosts = [
          host for host in zone_host_map[zone]
          if host_os_map[host] == os_version
      ]
      api.docker.run(
          image=_IMAGE_TEMPLATE % os_version,
          step_name='DHCP config test for %s on %s hosts: %s' %
          (zone, os_version, ', '.join(hosts)),
          cmd_args=[zone],
          dir_mapping=[(api.path['checkout'], '/src')])


def GenTests(api):

  def test_ufs_output(os_version):
    return api.json.output([{
        'name': _TEST_HOST,
        'osVersion': {
            'value': 'Linux Ubuntu %s' % os_version,
        }
    }])

  def changed_files(test_file='services/dhcpd/%s/foo' % _TEST_ZONE):
    t = api.override_step_data(
        'git diff to analyze patch', stdout=api.raw_io.output(test_file))
    t += api.path.exists(api.path['checkout'].join('chrome_golo', test_file))
    return t

  yield api.test(
      'chrome_golo_dhcp',
      api.properties(),
      api.buildbucket.try_build(),
      changed_files(),
      api.override_step_data(
          'get ufs host data', stdout=test_ufs_output(_IMAGE_VERSIONS[0])),
      api.override_step_data(
          'get ufs vm data', stdout=test_ufs_output(_IMAGE_VERSIONS[0])),
  )

  yield api.test(
      'chrome_golo_dhcp_no_dhcp_files_changed',
      api.properties(),
      api.buildbucket.try_build(),
      changed_files('not_a_dhcp_change'),
      api.post_process(post_process.StatusSuccess),
      api.post_process(post_process.DropExpectation),
  )

  yield api.test(
      'chrome_golo_dhcp_ufs_missing_hosts',
      api.properties(),
      api.buildbucket.try_build(),
      changed_files(),
      api.override_step_data('get ufs host data', stdout=api.json.output([])),
      api.override_step_data('get ufs vm data', stdout=api.json.output([])),
      api.post_process(post_process.StatusSuccess),
      api.post_process(post_process.DropExpectation),
  )

  yield api.test(
      'chrome_golo_dhcp_image_does_not_exist',
      api.properties(),
      api.buildbucket.try_build(),
      changed_files(),
      api.override_step_data(
          'get ufs host data', stdout=test_ufs_output(_IMAGE_VERSIONS[0])),
      api.override_step_data(
          'get ufs vm data', stdout=test_ufs_output(_IMAGE_VERSIONS[0])),
      api.override_step_data(
          'docker pull %s' % _IMAGE_TEMPLATE % _IMAGE_VERSIONS[0], retcode=1),
      api.post_process(post_process.StatusException),
      api.post_process(post_process.DropExpectation),
  )

  yield api.test(
      'chrome_golo_dhcp_non_float_os_version',
      api.properties(),
      api.buildbucket.try_build(),
      changed_files(),
      api.override_step_data(
          'get ufs host data', stdout=test_ufs_output('bad_version')),
      api.override_step_data(
          'get ufs vm data', stdout=test_ufs_output('bad_version')),
      api.post_process(post_process.StatusSuccess),
      api.post_process(post_process.DropExpectation),
  )

  yield api.test(
      'chrome_golo_dhcp_no_matching_image_for_os_version',
      api.properties(),
      api.buildbucket.try_build(),
      changed_files(),
      api.override_step_data(
          'get ufs host data', stdout=test_ufs_output('Linux Ubuntu 100.04')),
      api.override_step_data(
          'get ufs vm data', stdout=test_ufs_output('Linux Ubuntu 100.04')),
      api.post_process(post_process.StatusSuccess),
      api.post_process(post_process.DropExpectation),
  )
