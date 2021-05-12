# Copyright 2017 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import glob
import os
import platform
import shutil
import subprocess

from . import util

from .build_types import Wheel


class PlatformNotSupported(Exception):
  """Exception raised by Builder.build when the specified wheel's platform is
  not support."""


class Builder(object):

  def __init__(self, spec, arch_map=None, abi_map=None,
               only_plat=None, skip_plat=None):
    """Initializes a new wheel Builder.

    spec (Spec): The wheel specification.
    arch_map (dict or None): Naming map for architectures. If the current
        platform has an entry in this map, the generated wheel will use the
        value as the "platform" field.
    abi_map (dict or None): Naming map for ABI. If the current platform
        has an entry in this map, the generated wheel will use the
        value as the "abi" field.
    only_plat (iterable or None): If not None, this Builder will only declare
        that it can build for the named platforms.
    skip_plat (iterable or None): If not None, this Builder will avoid declaring
        that it can build for the named platforms.
    version_fn (callable or None): If not None, and spec.version is None, this
        function will be used to set the spec version at runtime.
    """
    if arch_map:
      assert not any(isinstance(v, basestring) for v in arch_map.values()), (
        'arch_map must map str->seq[str]')

    self._spec = spec
    self._arch_map = arch_map or {}
    self._abi_map = abi_map or {}
    self._only_plat = frozenset(only_plat or ())
    self._skip_plat = frozenset(skip_plat or ())

  def build_fn(self, system, wheel):
    """Must be overridden by the subclass.

    Args:
      system (runtime.System)
      wheel (types.Wheel) - The wheel we're attempting to build.

    Returns:
      None - The `wheel` argument will be uploaded in the CIPD package; the
        build_fn was expected to produce the wheel indicated by
        `wheel.filename`.
      list[types.Wheel] - This list of wheels will be uploaded in the CIPD
        package, instead of `wheel`.
    """
    raise NotImplementedError('Subclasses must implement build_fn')

  def version_fn(self, system):  # pylint: disable=unused-argument
    """Optionally overridden by the subclass.

    Returns:
      str - The version of the wheel to upload. By default this is
      `spec.version`.
    """
    return self._spec.version

  def md_data_fn(self):
    """Optionally overridden by the subclass.

    Returns:
      list[str] - Extra details to include in generated markdown files.
    """
    return []

  @property
  def spec(self):
    return self._spec

  def wheel(self, system, plat):
    spec = self.spec._replace(version=self.version_fn(system))

    # Make sure the Python version of the Wheel matches the platform.
    pyversion = plat.pyversion
    if spec.pyversions and pyversion not in spec.pyversions:
      # If the declared pyversions doesn't contain the version corresponding
      # to this platform, then we don't support it.
      raise PlatformNotSupported(
          ("Wheel %s specifies platform [%s] which has version [%s], but its "
           "pyversions '%r' doesn't contain this version") %
          (spec.name, plat.name, pyversion, spec.pyversions))

    wheel = Wheel(
        spec=spec,
        plat=plat,
        # e.g. cp27mu -> 27
        pyversion=plat.wheel_abi[2:4],
        filename=None,
        md_lines=self.md_data_fn())

    # Determine our package's wheel filename. This incorporates "abi" and "arch"
    # override maps, which are a priori knowledge of the package repository's
    # layout. This can differ from the local platform value if the package was
    # valid and built for multiple platforms, which seems to happen on Mac a
    # lot.
    plat_wheel = wheel._replace(
      plat=wheel.plat._replace(
        wheel_abi=self._abi_map.get(plat.name, plat.wheel_abi),
        wheel_plat=self._arch_map.get(plat.name, plat.wheel_plat),
      ),
    )
    return wheel._replace(
        filename=plat_wheel.default_filename(),
    )

  def supported(self, plat):
    if self._only_plat and plat.name not in self._only_plat:
      return False
    if plat.name in self._skip_plat:
      return False
    if not self.spec.universal:
      pyversion = plat.pyversion
      if self.spec.pyversions and pyversion not in self.spec.pyversions:
        return False

    return True

  def build(self, wheel, system, rebuild=False):
    if not self.supported(wheel.plat):
      raise PlatformNotSupported()

    pkg_path = os.path.join(system.pkg_dir, '%s.pkg' % (wheel.filename,))
    if not rebuild and os.path.isfile(pkg_path):
      util.LOGGER.info('Package is already built: %s', pkg_path)
      return pkg_path

    # Rebuild the wheel, if necessary. Get their ".whl" file paths.
    built_wheels = self.build_wheel(wheel, system, rebuild=rebuild)
    wheel_paths = [w.path(system) for w in built_wheels]

    # Create a CIPD package for the wheel. Give the wheel a universal filename
    # within the CIPD package.
    #
    # See "A Note on Universality" at the top.
    util.LOGGER.info('Creating CIPD package: %r => %r', wheel_paths, pkg_path)
    with system.temp_subdir('cipd_%s_%s' % wheel.spec.tuple) as tdir:
      for w in built_wheels:
        universal_wheel_path = os.path.join(tdir, w.universal_filename())
        shutil.copy(w.path(system), universal_wheel_path)
      _, git_revision = system.check_run(['git', 'rev-parse', 'HEAD'])
      system.cipd.create_package(wheel.cipd_package(git_revision),
                                 tdir, pkg_path)

    return pkg_path

  def build_wheel(self, wheel, system, rebuild=False):
    built_wheels = [wheel]
    wheel_path = wheel.path(system)
    if rebuild or not os.path.isfile(wheel_path):
      # The build_fn may return an alternate list of wheels.
      built_wheels = self.build_fn(system, wheel) or built_wheels
    else:
      util.LOGGER.info('Wheel is already built: %s', wheel_path)
    return built_wheels


def StageWheelForPackage(system, wheel_dir, wheel):
  """Finds the single wheel in wheel_dir and copies it to the filename indicated
  by wheel.filename.
  """
  # Find the wheel in "wheel_dir". We scan expecting exactly one wheel.
  wheels = glob.glob(os.path.join(wheel_dir, '*.whl'))
  assert len(wheels) == 1, 'Unexpected wheels: %s' % (wheels,)
  dst = os.path.join(system.wheel_dir, wheel.filename)

  source_path = wheels[0]
  util.LOGGER.debug('Identified source wheel: %s', source_path)
  shutil.copy(source_path, dst)


def EnvForWheel(wheel):
  """Returns any extra environment variables for building a wheel."""
  wheel_env = wheel.plat.env.copy()

  # If the wheel is python3, clear PYTHONPATH to avoid picking up
  # the default Python (2) modules.
  if wheel.spec.is_py3_only:
    wheel_env['PYTHONPATH'] = ''
  return wheel_env


def BuildPackageFromPyPiWheel(system, wheel):
  """Builds a wheel by obtaining a matching wheel from PyPi."""
  with system.temp_subdir('%s_%s' % wheel.spec.tuple) as tdir:
    util.check_run(
        system,
        None,
        tdir, [
            InstallCipdPythonPackage(system, wheel, tdir),
            '-m',
            'pip',
            'download',
            '--no-deps',
            '--only-binary=:all:',
            '--abi=%s' % (wheel.abi,),
            '--python-version=%s' % (wheel.pyversion,),
            '--platform=%s' % (wheel.primary_platform,),
            '%s==%s' % (wheel.spec.name, wheel.spec.version),
        ],
        cwd=tdir)

    StageWheelForPackage(system, tdir, wheel)

def HostCipdPlatform():
  """Return the CIPD platform for the host system.

  We need this to determine which Python CIPD package to download. We can't just
  use cipd_platform from the platform, because some platforms are
  cross-compiled.
  """

  ARM64_MACHINES = {'aarch64_be', 'aarch64', 'armv8b', 'armv8l'}
  system, machine = platform.system(), platform.machine()

  # Note that we don't need to match all possible values or combinations of
  # 'system' and 'machine', only the ones we actually have Python interpreters
  # for in CIPD.
  if system == 'Linux':
    if machine == 'x86_64':
      return 'linux-amd64'
    if machine in ARM64_MACHINES:
      return 'linux-arm64'
  elif system == 'Windows':
    if machine == 'AMD64':
      return 'windows-amd64'
    if machine in {'i386', 'i686'}:
      return 'windows-386'
  elif system == 'Darwin':
    if machine == 'x86_64':
      return 'mac-amd64'
    if machine in ARM64_MACHINES:
      return 'mac-arm64'

  raise Exception("No CIPD platform for %s, %s" % (system, machine))

def InstallCipdPythonPackage(system, wheel, base_dir):
  PY_CIPD_VERSION_MAP = {
    '27': 'version:2@2.7.18.chromium.34',
    '38': 'version:2@3.8.10.chromium.16',
    '39': 'version:2@3.9.5.chromium.16',
  }

  pkg_dir = os.path.join(base_dir, 'cipd_python%s_install' % (wheel.pyversion,))
  system.cipd.init(pkg_dir)

  cipd_pkg = 'infra/3pp/tools/cpython%s/%s' % (
      '3' if wheel.pyversion[0] == '3' else '', HostCipdPlatform())
  version = PY_CIPD_VERSION_MAP[wheel.pyversion]

  # Note that this will use the common CIPD instance cache in .dockerbuild, so
  # it will only download CPython once per run. It does however need to
  # extract the package, which takes a second or two. It could be worth
  # extracting to a common location and copying or symlinking into the temporary
  # workdir to save some time.
  system.cipd.install(cipd_pkg, version, pkg_dir)

  return os.path.join(pkg_dir, 'bin', 'python' + '.'.join(wheel.pyversion))

def BuildPackageFromSource(system, wheel, src, env=None):
  """Creates Python wheel from src.

  Args:
    system (dockerbuild.runtime.System): Represents the local system.
    wheel (dockerbuild.wheel.Wheel): The wheel to build.
    src (dockerbuild.source.Source): The source to build the wheel from.
    env (Dict[str, str]|None): Additional envvars to set while building the
      wheel.
  """
  dx = system.dockcross_image(wheel.plat)
  with system.temp_subdir('%s_%s' % wheel.spec.tuple) as tdir:
    build_dir = system.repo.ensure(src, tdir)

    for patch in src.get_patches():
      util.LOGGER.info('Applying patch %s', os.path.basename(patch))
      cmd = ['patch', '-p1', '--quiet', '-i', patch]
      subprocess.check_call(cmd, cwd=build_dir)

    cmd = [
        InstallCipdPythonPackage(system, wheel, build_dir),
        '-m',
        'pip',
        'wheel',
        '--no-deps',
        '--only-binary=:all:',
        '--wheel-dir',
        tdir,
        '.',
    ]

    extra_env = EnvForWheel(wheel)
    if dx.platform:
      extra_env.update({
          'host_alias': dx.platform.cross_triple,
      })
    if env:
      extra_env.update(env)

    util.check_run(
        system,
        dx,
        tdir,
        cmd,
        cwd=build_dir,
        env=extra_env)

    StageWheelForPackage(system, tdir, wheel)
