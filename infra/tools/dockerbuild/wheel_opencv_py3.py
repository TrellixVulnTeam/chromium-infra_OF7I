# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import re
import os
import subprocess

from . import build_platform
from . import util
from . import source
from . import wheel_wheel

from .builder import Builder
from .builder import BuildPackageFromPyPiWheel
from .builder import HostCipdPlatform
from .builder import SetupPythonPackages
from .builder import StageWheelForPackage

from .build_types import Spec


class OpenCVPy3(Builder):

  def __init__(self,
               name,
               version,
               numpy_version,
               pyversions=None,
               default=True,
               patches=(),
               patch_base=None,
               patch_version=None,
               **kwargs):
    self._pypi_src = source.pypi_sdist(
        name, version, patches=patches, patch_base=patch_base)
    self._packaged = set(
        kwargs.pop('packaged', (p.name for p in build_platform.PACKAGED)))
    self._env = kwargs.pop('env', None)
    version_suffix = '.' + patch_version if patch_version else None

    self._numpy_version = numpy_version

    super(OpenCVPy3, self).__init__(
        Spec(
            name,
            self._pypi_src.version,
            universal=False,
            pyversions=pyversions,
            default=default,
            version_suffix=version_suffix), **kwargs)

  def build_fn(self, system, wheel):
    if wheel.plat.name in self._packaged:
      return BuildPackageFromPyPiWheel(system, wheel)
    return BuildPackageFromSource(system, wheel, self._numpy_version,
                                  self._pypi_src, self._env)


def BuildPackageFromSource(system, wheel, numpy_version, src, env=None):
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
    # Workaround for windows to avoid file paths exceeding 260 limit.
    # These samples are not required for building opencv.
    sample_re = re.compile('opencv-python-[0-9.]+/opencv/samples/.*')
    file_filter = lambda path: not sample_re.match(path)
    build_dir = system.repo.ensure(src, tdir, unpack_file_filter=file_filter)
    util.copy_to(util.resource_path('generate_pyproject.py'), build_dir)

    for patch in src.get_patches():
      util.LOGGER.info('Applying patch %s', os.path.basename(patch))
      cmd = ['git', 'apply', '-p1', patch]
      subprocess.check_call(cmd, cwd=build_dir)

    numpy_builder = wheel_wheel.SourceOrPrebuilt('numpy', numpy_version)
    numpy_wheel = numpy_builder.wheel(system, wheel.plat)
    numpy_builder.build(numpy_wheel, system)
    numpy_path = numpy_wheel.path(system)
    numpy_path = util.copy_to(numpy_path, build_dir)

    interpreter, extra_env = SetupPythonPackages(system, wheel, build_dir)

    # Generate pyproject.toml to control build dependencies
    cmd = [
        interpreter,
        'generate_pyproject.py',
        '--remotes',
        'setuptools',
        'wheel',
        'scikit-build',
        'cmake',
        'pip',
        '--locals',
        'numpy@{}'.format(os.path.basename(numpy_path)),
    ]
    util.check_run(system, dx, tdir, cmd, cwd=build_dir, env=extra_env)

    # Build wheel
    cmd = [
        interpreter,
        '-m',
        'pip',
        'wheel',
        '--no-deps',
        '--only-binary=:all:',
        '--wheel-dir',
        tdir,
        '.',
    ]

    extra_env = {}
    if dx.platform:
      extra_env.update({
          'host_alias': dx.platform.cross_triple,
      })
    if env:
      extra_env.update(env)

    util.check_run(system, dx, tdir, cmd, cwd=build_dir, env=extra_env)

    StageWheelForPackage(system, tdir, wheel)
