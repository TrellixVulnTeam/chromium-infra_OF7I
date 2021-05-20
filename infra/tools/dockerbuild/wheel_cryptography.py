# Copyright 2017 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import os

from .build_types import Spec
from .builder import (Builder, BuildPackageFromPyPiWheel, StageWheelForPackage,
                      SetupPythonPackages)

from . import source
from . import util

class Cryptography(Builder):

  def __init__(self,
               name,
               crypt_src,
               openssl_src,
               packaged=None,
               pyversions=None,
               **kwargs):
    """Specialized wheel builder for the "cryptography" package.

    Args:
      name (str): The wheel name.
      crypt_src (Source): The Source for the cryptography package. The wheel
          version will be extracted from this.
      openssl_src (Source): The OpenSSL source to build against.
      packaged (iterable or None): The names of platforms that have this wheel
          available via PyPi. If None, a default set of packaged wheels will be
          generated based on standard PyPi expectations, encoded with each
          Platform's "packaged" property.
      pyversions: (See Buidler's "pyversions" argument.)
    """
    self._packaged = packaged or ()
    self._crypt_src = crypt_src
    self._openssl_src = openssl_src
    super(Cryptography, self).__init__(
        Spec(
            name,
            crypt_src.version,
            universal=False,
            pyversions=pyversions,
            default=True), **kwargs)

  def build_fn(self, system, wheel):
    if wheel.plat.name in self._packaged:
      return BuildPackageFromPyPiWheel(system, wheel)

    dx = system.dockcross_image(wheel.plat)
    assert dx, 'Docker image required for compilation.'
    with system.temp_subdir('%s_%s' % wheel.spec.tuple) as tdir:
      # Unpack "cryptography".
      crypt_dir = system.repo.ensure(self._crypt_src, tdir)

      # Unpack "OpenSSL" into the "openssl/" subdirectory.
      openssl_dir = system.repo.ensure(self._openssl_src, tdir)

      # Build OpenSSL. We build this out of "openssl_dir" and install to
      # <openssl_dir>/PREFIX, so that will be the on-disk path to our OpenSSL
      # libraries.
      #
      # "Configure" must be run in the directory in which it builds, so we
      # `cd` into "openssl_dir" using dockcross "run_args".
      prefix = dx.workrel(tdir, tdir, 'prefix')
      util.check_run_script(
          system,
          dx,
          tdir,
          [
              '#!/bin/bash',
              'set -e',
              'export NUM_CPU="$(getconf _NPROCESSORS_ONLN)"',
              'echo "Using ${NUM_CPU} CPU(s)"',
              ' '.join([
                  './Configure',
                  '-fPIC',
                  '--prefix=%s' % (prefix,),
                  # We already pass the full path in CC etc variables
                  '--cross-compile-prefix=',
                  'no-shared',
                  # https://github.com/openssl/openssl/issues/1685
                  'no-afalgeng',
                  'no-ssl3',
                  wheel.plat.openssl_target,
              ]),
              'make -j${NUM_CPU}',
              'make install',
          ],
          cwd=openssl_dir,
      )

      py_binary, env = SetupPythonPackages(system, wheel, tdir)
      py_binary = dx.workrel(tdir, py_binary)

      # Dockcross containers already contain cffi installed on the system.
      # For other platforms, we run the setup.py script under vpython (assumed
      # to be present in $PATH, i.e. in Swarming or via depot_tools), so we can
      # pre-install this wheel and its dependencies.
      if wheel.plat.dockcross_base is None:
        interpreter_flag = '-vpython-interpreter %s' % py_binary
        if wheel.plat.pyversion == 'py2':
          py_binary = 'vpython'
          vpython_spec = '.vpython'
        else:
          py_binary = 'vpython3'
          vpython_spec = '.vpython3'
        py_binary += ' %s' % interpreter_flag

        with open(os.path.join(crypt_dir, vpython_spec), 'w') as spec:
          for name, version in [('cffi/${vpython_platform}', '1.14.5'),
                                ('pycparser-py2_py3', '2.17')]:
            spec.write('wheel: <\n')
            spec.write('  name: "infra/python/wheels/%s"\n' % name)
            spec.write('  version: "version:%s"\n' % version)
            spec.write('>\n')

      # Build "cryptography".
      d = {
        'prefix': prefix,
      }

      util.check_run_script(
          system,
          dx,
          tdir,
          [
              '#!/bin/bash',
              'set -e',
              'export CFLAGS="' + ' '.join([
                  '-I%(prefix)s/include' % d,
                  '$CFLAGS',
              ]) + '"',
              'export LDFLAGS="' + ' '.join([
                  '-L%(prefix)s/lib' % d,
                  '-L%(prefix)s/lib64' % d,
                  '$LDFLAGS',
              ]) + '"',
              ' '.join([
                  py_binary,
                  'setup.py',
                  'build_ext',
                  '--include-dirs',
                  '/usr/cross/include',
                  '--library-dirs',
                  '/usr/cross/lib',
                  '--force',
                  'build',
                  '--force',
                  'build_scripts',
                  '--executable=/usr/local/bin/python',
                  '--force',
                  'bdist_wheel',
                  '--plat-name',
                  wheel.primary_platform,
              ]),
          ],
          cwd=crypt_dir,
          env=env,
      )

      StageWheelForPackage(
        system, os.path.join(crypt_dir, 'dist'), wheel)
      return None


class CryptographyPyPI(Cryptography):

  def __init__(self, name, ver, openssl, pyversions=('py2',), **kwargs):
    """Adapts Cryptography wheel builder to use available PyPI wheels.

    Builds wheels for platforms not present in PyPI (e.g ARM) from source.
    Builds statically and links to OpenSSL of given version.
    """
    super(CryptographyPyPI, self).__init__(
        name,
        source.pypi_sdist('cryptography', ver),
        source.remote_archive(
            name='openssl',
            version=openssl,
            url='https://www.openssl.org/source/openssl-%s.tar.gz' % openssl,
        ),
        arch_map={
            'mac-x64': ['macosx_10_6_intel'],
        },
        pyversions=pyversions,
        **kwargs)
