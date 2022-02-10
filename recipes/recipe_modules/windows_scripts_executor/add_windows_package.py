# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

INSTALL_PACKAGE_CMD = 'Add-WindowsPackage'
INSTALL_PACKAGE_SCRIPT = 'Add-WindowsPackage.ps1'
INSTALL_PACKAGE_PATH = '-PackagePath {}'
INSTALL_PACKAGE_ROOT = '-Path {}'
INSTALL_PACKAGE_LOG_PATH = '-LogPath "{}"'
INSTALL_PACKAGE_LOG_LEVEL = '-LogLevel {}'


def install_package(powershell,
                    scripts,
                    awp,
                    package,
                    mnt_dir,
                    logs,
                    log_level='WarningsInfo'):
  """Install a package to the mounted image"""
  # Args for the install command
  args = [
      INSTALL_PACKAGE_PATH.format(package),
      INSTALL_PACKAGE_ROOT.format(mnt_dir),
      INSTALL_PACKAGE_LOG_PATH.format(
          logs.join(INSTALL_PACKAGE_CMD, 'ins_pkg.log')),
      INSTALL_PACKAGE_LOG_LEVEL.format(log_level)
  ]

  for a in awp.args:
    args.append(a)

  return powershell(
      'Install package {}'.format(awp.name),
      scripts.join(INSTALL_PACKAGE_SCRIPT),
      logs=[logs.join(INSTALL_PACKAGE_CMD)],
      args=args)
