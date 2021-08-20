# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Format strings for use in mount cmdline options
MOUNT_CMD = 'Mount-WindowsImage'
MOUNT_IMG_FILE = '-ImagePath "{}"'
MOUNT_INDEX = '-Index {}'
MOUNT_DIR = '-Path "{}"'
MOUNT_LOG_PATH = '-LogPath "{}"'
MOUNT_LOG_LEVEL = '-LogLevel {}'


def mount_win_wim(powershell,
                  mnt_dir,
                  image,
                  index,
                  logs,
                  log_level='WarningsInfo'):
  """Mount the wim to a dir for modification"""
  # Args for the mount cmd
  args = [
      MOUNT_IMG_FILE.format(image),
      MOUNT_INDEX.format(index),
      MOUNT_DIR.format(mnt_dir),
      MOUNT_LOG_PATH.format(logs.join(MOUNT_CMD, 'mount.log')),  # include logs
      MOUNT_LOG_LEVEL.format(log_level)
  ]
  powershell(
      'Mount wim to {}'.format(mnt_dir),
      MOUNT_CMD,
      logs=[logs.join(MOUNT_CMD)],
      args=args)
