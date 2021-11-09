# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

INSTALL_DRIVER_CMD = 'Add-WindowsDriver'
INSTALL_DRIVER_PATH = '-Driver {}'
INSTALL_DRIVER_ROOT = '-Path {}'
INSTALL_DRIVER_LOG_PATH = '-LogPath "{}"'
INSTALL_DRIVER_LOG_LEVEL = '-LogLevel {}'


def install_driver(powershell,
                   awd,
                   package,
                   mnt_dir,
                   logs,
                   log_level='WarningsInfo'):
  """Install a driver to the mounted image"""
  # Args for the install command
  args = [
      INSTALL_DRIVER_PATH.format(package),
      INSTALL_DRIVER_ROOT.format(mnt_dir),
      INSTALL_DRIVER_LOG_PATH.format(
          logs.join(INSTALL_DRIVER_CMD, 'ins_driver.log')),
      INSTALL_DRIVER_LOG_LEVEL.format(log_level)
  ]

  for a in awd.args:
    args.append(a)

  return powershell(
      'Install driver {}'.format(awd.name),
      INSTALL_DRIVER_CMD,
      logs=[logs.join(INSTALL_DRIVER_CMD)],
      args=args)
