# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import collections
import os

from . import cipd

# BINARY_VERSION_SUFFIX is a string added to the end of each version tag. This
# can be used to distinguish one build of a given package from another.
#
# Incrementing BINARY_VERSION only affects binary wheels; it is not applied to
# universal wheels. Changing this is a heavy operation, requiring the user to
# regenerate all wheels for all platforms so that they become available with the
# new suffix.
BINARY_VERSION_SUFFIX = None

_Spec = collections.namedtuple(
    '_Spec',
    (
        'name',
        'version',
        # True if the wheel is universal, that is, pure Python.
        'universal',
        # An iterable containing the major Python versions the wheel is
        # expected to be compatible with. Valid values are ["py2", "py3"].
        'pyversions',
        # default is true if this Spec should be built by default (i.e., when a
        # user doesn't manually specify Specs to build).
        'default',
        # If set, this is appended to the CIPD version tag. It must include any
        # separator.
        'version_suffix'))


class Spec(_Spec):

  @property
  def tuple(self):
    return (self.name, self.version)

  @property
  def tag(self):
    if self.version:
      ret = '%s-%s' % (self.name, self.version)
    else:
      ret = self.name
    if self.version_suffix:
      ret += self.version_suffix
    if self.pyversions:
      ret += '-%s' % '.'.join(sorted(self.pyversions))
    return ret

  @property
  def is_py3_only(self):
    return self.pyversions == ['py3']

  def to_universal(self):
    return self._replace(universal=True)


_Wheel = collections.namedtuple(
    '_Wheel', ('spec', 'plat', 'pyversion', 'filename', 'md_lines'))


class Wheel(_Wheel):

  def __new__(cls, *args, **kwargs):
    kwargs.setdefault('md_lines', [])
    return super(Wheel, cls).__new__(cls, *args, **kwargs)

  @property
  def pyversion_str(self):
    if self.spec.universal:
      # We support py2-only, py3-only, or py2+py3.  Wait for other requests
      # to show up before adding more.
      pyv = sorted(self.spec.pyversions or ['py2', 'py3'])
      if pyv == ['py2', 'py3']:
        return 'py2.py3'
      elif pyv == ['py2'] or pyv == ['py3']:
        return pyv[0]
      else:
        raise ValueError('Unsupported versions: %r' % (pyv,))

    # We only generate wheels for "cpython" at the moment, and only
    # for the specific wheel ABI the platform is configured for.
    return 'cp%s' % (self.pyversion,)

  @property
  def abi(self):
    if self.spec.universal or not self.plat.wheel_abi:
      return 'none'
    return self.plat.wheel_abi

  @property
  def platform(self):
    return ['any'] if self.spec.universal else self.plat.wheel_plat

  @property
  def primary_platform(self):
    """The platform to use when naming intermediate wheels and requesting
    wheel from "pip". Generally, platforms that this doesn't work on (e.g.,
    ARM) will not have wheels in PyPi, and platforms with wheels in
    PyPi will have only one platform.

    This is also used for naming when building wheels; this choice is
    inconsequential in this context, as the wheel is renamed after the build.
    """
    return self.platform[0]

  @property
  def build_id(self):
    """Returns a unique identifier for this build of the wheel."""
    build_id = self.spec.version
    if self.spec.version_suffix:
      build_id += self.spec.version_suffix
    return build_id

  def default_filename(self):
    return '%(name)s-%(version)s-%(pyversion)s-%(abi)s-%(platform)s.whl' % {
        'name': self.spec.name.replace('-', '_'),
        'version': self.spec.version,
        'pyversion': self.pyversion_str,
        'abi': self.abi,
        'platform': '.'.join(self.platform),
    }

  def universal_filename(self):
    """This is a universal filename for the wheel, regardless of whether it's
    binary or truly universal. See "A Note on Universality" at the top for
    details on why we'd ever want to do this.
    """
    wheel = self._replace(spec=self.spec.to_universal())
    return wheel.default_filename()

  def path(self, system):
    return os.path.join(system.wheel_dir, self.filename)

  def cipd_package(self, git_revision=None, templated=False):
    base_path = ['infra', 'python', 'wheels']
    if self.spec.universal:
      base_path += ['%s-%s' % (self.spec.name, self.pyversion_str)]
    else:
      base_path += [self.spec.name]
      if not templated:
        base_path += [
            '%s_%s_%s' % (self.plat.cipd_platform, self.pyversion_str, self.abi)
        ]
      else:
        base_path += ['${vpython_platform}']

    version_tag = 'version:%s' % (self.build_id,)
    if not self.spec.universal and BINARY_VERSION_SUFFIX:
      version_tag += BINARY_VERSION_SUFFIX
    tags = [version_tag]

    if git_revision is not None:
      tags.append('git_revision:%s' % (git_revision,))

    base = self.plat.dockcross_base
    if base is not None and 'manylinux' in base:
      tags.append('manylinux_version:%s' % (base,))

    return cipd.Package(
        name=('/'.join(p.replace('.', '_') for p in base_path)).lower(),
        tags=tuple(tags),
        install_mode=cipd.INSTALL_SYMLINK,
        compress_level=cipd.COMPRESS_NONE,
    )
