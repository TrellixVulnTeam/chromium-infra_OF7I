# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Implements the main creation logic.

This defines the encapsulating logic for fetching, building, packaging, testing
and uploading a ResolvedSpec.
"""
import re

from . import source
from . import build
from . import resolved_spec
from . import verify

from .workdir import Workdir

from PB.recipe_modules.infra.support_3pp.spec import Spec


def build_resolved_spec(api, spec_lookup, cache, force_build, spec, version,
                        ecosystem_hash):
  """Builds a resolved spec at a specific version, then uploads it.

  Args:
    * api - The ThirdPartyPackagesNGApi's `self.m` module collection.
    * spec_lookup ((package_name, platform) -> ResolvedSpec) - A function to
      lookup (possibly cached) ResolvedSpec's for things like dependencies and
      tools.
    * cache (dict) - A map of (package_name, version, platform) -> CIPDSpec.
      The `build_resolved_spec` function fully manages the content of this
      dictionary.
    * force_build (bool) - If True, don't consult CIPD server to see if the
      package is already built. This also disables uploading the source and
      built results, to avoid attempting to upload a duplicately-tagged package.
    * spec (ResolvedSpec) - The resolved spec to build.
    * version (str) - The symver (or 'latest') version of the package to build.
    * ecosystem_hash(str) - If specified, tells 3pp hash used for this build.

  Returns the CIPDSpec of the built package; If the package already existed on
  the remote server, it will return the CIPDSpec immediately (without attempting
  to build anything).
  """
  keys = [(spec.cipd_pkg_name, version, spec.platform)]
  if keys[0] in cache:
    return cache[keys[0]]

  def set_cache(spec):
    for k in keys:
      cache[k] = spec
    return spec

  with api.step.nest('building %s' % (spec.cipd_pkg_name,)):
    env = {
      '_3PP_PLATFORM': spec.platform,
      '_3PP_TOOL_PLATFORM': spec.tool_platform,
      '_3PP_CIPD_PACKAGE_NAME': spec.cipd_pkg_name,
      # CIPD uses 'mac' instead of 'darwin' for historical reasons.
      'GOOS': spec.platform.split('-')[0].replace('mac', 'darwin'),
      # CIPD encodes the GOARCH/GOARM pair of ('arm', '6') as 'armv6l'.
      # Since GOARCH=6 is the default, we don't need to specify it.
      'GOARCH': spec.platform.split('-')[1].replace('armv6l', 'arm'),
    }
    if spec.platform.startswith('mac-'):
      if spec.platform == 'mac-arm64':
        # ARM64 support is added in macOS 11.
        env['MACOSX_DEPLOYMENT_TARGET'] = '11.0'

        # Mac builds don't use Docker/Dockcross, so we handle cross-build
        # setup here. Setting CCC_OVERRIDE_OPTIONS passes the target to
        # Clang globally, so we don't need to plumb it through each individual
        # install script. We use '^' to indicate this option is inserted at
        # the beginning of the compiler options list -- this gives the ability
        # to override it later if needed.
        if resolved_spec.platform_for_host(api) != spec.platform:
          env['CROSS_TRIPLE'] = 'aarch64-apple-darwin'
          env['CCC_OVERRIDE_OPTIONS'] = '^--target=arm64-apple-macos'
        else:
          # Make sure to clear these options if not cross-compiling, since
          # we may be switching back and forth between building host tools
          # and target-platform binaries.
          env.pop('CROSS_TRIPLE', None)
          env.pop('CCC_OVERRIDE_OPTIONS', None)
      else:
        env['MACOSX_DEPLOYMENT_TARGET'] = '10.10'
    if spec.create_pb.source.patch_version:
      env['_3PP_PATCH_VERSION'] = spec.create_pb.source.patch_version

    with api.context(env=env):
      # Resolve 'latest' versions. Done inside the env because 'script' based
      # sources need the $_3PP* envvars.
      is_latest = version == 'latest'
      git_hash = ''
      if is_latest:
        version, git_hash = source.resolve_latest(api, spec)
        keys.append((spec.cipd_pkg_name, version, spec.platform))
        if keys[-1] in cache:
          return set_cache(cache[keys[-1]])

      cipd_spec = spec.cipd_spec(version)
      # See if the specific version is uploaded
      if force_build or not cipd_spec.exists_in_cipd():
        # Otherwise, build it
        _build_impl(
            api, cipd_spec, is_latest, spec_lookup, force_build,
            (lambda spec, version: build_resolved_spec(
                api, spec_lookup, cache, force_build, spec, version,
                ecosystem_hash)),
            spec, version, git_hash, ecosystem_hash)

      return set_cache(cipd_spec)


def _build_impl(api, cipd_spec, is_latest, spec_lookup, force_build, recurse_fn,
                spec, version, git_hash, ecosystem_hash):
  workdir = Workdir(api, spec, version)
  with api.context(env={'_3PP_VERSION': version}):
    api.file.ensure_directory('mkdir -p [workdir]', workdir.base)

    with api.step.nest('fetch sources'):
      source.fetch_source(
          api,
          workdir,
          spec,
          version,
          git_hash,
          spec_lookup,
          recurse_fn,
          skip_upload=force_build)

    if spec.create_pb.HasField("build"):
      with api.step.nest('run installation'):
        build.run_installation(api, workdir, spec)
      installed_prefix = workdir.output_prefix
    else:
      installed_prefix = workdir.checkout

    # List files to be packaged (for debugging).
    api.file.listdir(
        'List files to be packaged', installed_prefix, recursive=True)

    # Package stage
    cipd_spec.build(installed_prefix,
                    Spec.Create.Package.InstallMode.Name(
                      spec.create_pb.package.install_mode),
                    spec.create_pb.package.version_file)

    if spec.create_pb.HasField("verify"):
      with api.step.nest('run test'):
        verify.run_test(api, workdir, spec, cipd_spec)

    if not force_build:
      with api.step.nest('do upload') as upload_presentation:
        extra_tags = {'3pp_ecosystem_hash': ecosystem_hash}
        if spec.create_pb.package.alter_version_re:
          extra_tags['real_version'] = version
        cipd_spec.ensure_uploaded(is_latest and not spec.disable_latest_ref,
                                  extra_tags)

        # the active_result could be from cipd.describe or cipd.register
        upload_step_result = api.step.active_result
        if upload_step_result.json.output:
          pin_result = upload_step_result.json.output['result']
          # When from cipd.describe, 'instance_id' and 'package' will be
          # under the key 'pin'
          if 'pin' in pin_result:
            pin_result = pin_result['pin']
          if 'instance_id' in pin_result and 'package' in pin_result:
            upload_presentation.links[pin_result['instance_id']] = (
                'https://chrome-infra-packages.appspot.com' +
                '/p/%(package)s/+/%(instance_id)s' % pin_result)

        # If reporting to Snoopy is enabled, try to report built package.
        if 'security.snoopy' in api.buildbucket.build.input.experiments:
          # Attach provenance after the package has been uploaded.
          package_hash = api.file.file_hash(cipd_spec.local_pkg_path(),
                                            test_data='deadbeef')
          try:
            api.snoopy.report_cipd(package_hash, cipd_spec.name,
                                  pin_result['instance_id'])
          except Exception:  # pragma: no cover
            api.step.active_result.presentation.status = api.step.FAILURE
