# Copyright 2017 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import collections
import platform
import sys


class Platform(
    collections.namedtuple(
        'Platform',
        (
            # The name of the platform.
            'name',

            # If the platform is "manylinux', the "manylinux" Docker image build
            # name (e.g., "cp27-cp27mu").
            'manylinux_name',

            # The value to pass to e.g. `./configure --host ...`
            'cross_triple',

            # The Python wheel ABI.
            'wheel_abi',

            # Tuple of Python wheel platforms. Must have at least one.
            #
            # This is used for local wheel naming. Wheels are named universally
            # within CIPD packages. Changing this will not impact CIPD package
            # contents, but will affect the locally generated intermediate wheel
            # names.
            'wheel_plat',

            # The "dockcross" base image (can be None).
            'dockcross_base',

            # The OpenSSL "Configure" script build target.
            'openssl_target',

            # Do Python wheels get packaged on PyPi for this platform?
            'packaged',

            # The name of the CIPD platform to use.
            'cipd_platform',

            # Extra environment variables to set when building wheels on this
            # platform.
            'env',
        ))):

  @property
  def dockcross_base_image(self):
    if not self.dockcross_base:
      return None
    return 'dockcross/%s' % (self.dockcross_base,)

  @property
  def dockcross_image_tag(self):
    return 'infra-dockerbuild-%s' % (self.name,)

  @property
  def pyversion(self):
    return 'py2' if self.wheel_abi.startswith('cp2') else 'py3'


ALL = {
    p.name: p for p in (
        Platform(
            name='linux-armv6',
            manylinux_name=None,
            cross_triple='arm-linux-gnueabihf',
            wheel_abi='cp27mu',
            wheel_plat=('linux_armv6l', 'linux_armv7l', 'linux_armv8l',
                        'linux_armv9l'),
            dockcross_base='linux-armv6',
            openssl_target='linux-armv4',
            packaged=False,
            cipd_platform='linux-armv6l',
            env={},
        ),
        Platform(
            name='linux-arm64',
            manylinux_name=None,
            cross_triple='aarch64-unknown-linux-gnueabi',
            wheel_abi='cp27mu',
            wheel_plat=('linux_arm64', 'linux_aarch64'),
            dockcross_base='linux-arm64',
            openssl_target='linux-aarch64',
            packaged=False,
            cipd_platform='linux-arm64',
            env={},
        ),
        Platform(
            name='linux-arm64-py3',
            manylinux_name=None,
            cross_triple='aarch64-unknown-linux-gnueabi',
            wheel_abi='cp38',
            wheel_plat=('linux_arm64', 'linux_aarch64'),
            dockcross_base='linux-arm64',
            openssl_target='linux-aarch64',
            packaged=False,
            cipd_platform='linux-arm64',
            env={},
        ),
        Platform(
            name='linux-mipsel',
            manylinux_name=None,
            cross_triple='mipsel-linux-gnu',
            wheel_abi='cp27mu',
            wheel_plat=('linux_mipsel',),
            dockcross_base='linux-mipsel',
            openssl_target='linux-mips32',
            packaged=False,
            cipd_platform='linux-mips32',
            env={},
        ),

        # NOTE: "mips" and "mips64" both use 32-bit MIPS, which is consistent
        # with our devices.
        Platform(
            name='linux-mips',
            manylinux_name=None,
            cross_triple='mips-unknown-linux-gnu',
            wheel_abi='cp27mu',
            wheel_plat=('linux_mips',),
            dockcross_base='linux-mips',
            openssl_target='linux-mips32',
            packaged=False,
            cipd_platform='linux-mips',
            env={},
        ),
        Platform(
            name='linux-mips64',
            manylinux_name=None,
            cross_triple='mips-unknown-linux-gnu',
            wheel_abi='cp27mu',
            wheel_plat=('linux_mips64',),
            dockcross_base='linux-mips',
            openssl_target='linux-mips32',
            packaged=False,
            cipd_platform='linux-mips64',
            env={},
        ),
        Platform(
            name='manylinux-x64',
            manylinux_name='cp27-cp27mu',
            cross_triple='x86_64-linux-gnu',
            wheel_abi='cp27mu',
            wheel_plat=('manylinux2014_x86_64',),
            dockcross_base='manylinux2014-x64',
            openssl_target='linux-x86_64',
            packaged=True,
            cipd_platform='linux-amd64',
            env={},
        ),
        Platform(
            name='manylinux-x64-py3',
            manylinux_name=None,  # Don't use any built-in Python
            cross_triple='x86_64-linux-gnu',
            wheel_abi='cp38',
            wheel_plat=('manylinux2014_x86_64',),
            dockcross_base='manylinux2014-x64',
            openssl_target='linux-x86_64',
            packaged=True,
            cipd_platform='linux-amd64',
            env={},
        ),

        # Atypical but possible Python configuration using "2-byte UCS", with
        # ABI "cp27m".
        Platform(
            name='manylinux-x64-ucs2',
            manylinux_name='cp27-cp27m',
            cross_triple='x86_64-linux-gnu',
            wheel_abi='cp27m',
            wheel_plat=('manylinux2014_x86_64',),
            dockcross_base='manylinux2014-x64',
            openssl_target='linux-x86_64',
            packaged=True,
            cipd_platform='linux-amd64',
            env={},
        ),
        Platform(
            name='mac-x64',
            manylinux_name=None,
            cross_triple='',
            wheel_abi='cp27m',
            wheel_plat=('macosx_10_11_x86_64',),
            dockcross_base=None,
            openssl_target='darwin64-x86_64-cc',
            packaged=True,
            cipd_platform='mac-amd64',
            # This ensures compatibibility regardless of the OS version this
            # runs on.
            env={'MACOSX_DEPLOYMENT_TARGET': '10.11'},
        ),
        Platform(
            # TODO: Rename to -py3 to conform to other Python 3 platform names.
            name='mac-x64-cp38',
            manylinux_name=None,
            cross_triple='',
            wheel_abi='cp38',
            wheel_plat=('macosx_10_11_x86_64',),
            dockcross_base=None,
            openssl_target='darwin64-x86_64-cc',
            packaged=True,
            cipd_platform='mac-amd64',
            env={
                # Necessary for some wheels to build. See for instance:
                # https://github.com/giampaolo/psutil/issues/1832
                'ARCHFLAGS': '-arch x86_64',
                'MACOSX_DEPLOYMENT_TARGET': '10.11'
            },
        ),
        Platform(
            # TODO: Remove once we have bootstrapped native builders.
            name='mac-arm64-cp38-cross',
            manylinux_name=None,
            cross_triple='',
            wheel_abi='cp38',
            wheel_plat=('macosx_11_0_arm64',),
            dockcross_base=None,
            openssl_target='darwin64-arm64-cc',
            # We've done our own backport of ARM64 support to python 3.8, so
            # there won't be any pre-packaged wheels available.
            packaged=False,
            cipd_platform='mac-arm64',
            env={
                # Necessary for some wheels to build. See for instance:
                # https://github.com/giampaolo/psutil/issues/1832
                'ARCHFLAGS': '-arch arm64',
                # Setting CCC_OVERRIDE_OPTIONS in this way makes clang work
                # similar to a dockcross cross-compiler, and is the most robust
                # mechanism to deal with wheels' varying setup.py
                # implementations.
                'CCC_OVERRIDE_OPTIONS': '+--target=arm64-apple-macos',
                'MACOSX_DEPLOYMENT_TARGET': '11.0'
            },
        ),
        Platform(
            name='windows-x86',
            manylinux_name=None,
            cross_triple='',
            wheel_abi='cp27m',
            wheel_plat=('win32',),
            dockcross_base=None,
            openssl_target='Cygwin-x86',
            packaged=True,
            cipd_platform='windows-386',
            env={},
        ),
        Platform(
            name='windows-x86-py3',
            manylinux_name=None,
            cross_triple='',
            wheel_abi='cp38',
            wheel_plat=('win32',),
            dockcross_base=None,
            openssl_target='Cygwin-x86',
            packaged=True,
            cipd_platform='windows-386',
            env={},
        ),
        Platform(
            name='windows-x64',
            manylinux_name=None,
            cross_triple='',
            wheel_abi='cp27m',
            wheel_plat=('win_amd64',),
            dockcross_base=None,
            openssl_target='Cygwin-x86_64',
            packaged=True,
            cipd_platform='windows-amd64',
            env={},
        ),
        Platform(
            name='windows-x64-py3',
            manylinux_name=None,
            cross_triple='',
            wheel_abi='cp38',
            wheel_plat=('win_amd64',),
            dockcross_base=None,
            openssl_target='Cygwin-x86_64',
            packaged=True,
            cipd_platform='windows-amd64',
            env={},
        ),
    )
}
NAMES = sorted(ALL.keys())
PACKAGED = [p for p in ALL.itervalues() if p.packaged]
ALL_LINUX = [p.name for p in ALL.itervalues() if 'linux' in p.name]


def NativePlatforms():
  # Identify our native platforms.
  if sys.platform == 'darwin':
    if platform.machine() == 'x86_64':
      return [ALL['mac-x64'], ALL['mac-x64-cp38'], ALL['mac-arm64-cp38-cross']]
    # TODO: Native ARM64 platform.
  elif sys.platform == 'win32':
    return [
        ALL['windows-x86'], ALL['windows-x86-py3'], ALL['windows-x64'],
        ALL['windows-x64-py3']
    ]
  elif sys.platform == 'linux2':
    # Linux platforms are built with docker, so Linux doesn't support any
    # platforms natively.
    return []
  raise ValueError('Cannot identify native image for %r-%r.' %
                   (sys.platform, platform.machine()))


# Represents the "universal package" platform.
UNIVERSAL = Platform(
    name='universal',
    manylinux_name=None,
    cross_triple='',
    wheel_abi='none',
    wheel_plat=('any',),
    dockcross_base=None,
    openssl_target=None,
    packaged=True,
    cipd_platform=None,
    env={},
)
