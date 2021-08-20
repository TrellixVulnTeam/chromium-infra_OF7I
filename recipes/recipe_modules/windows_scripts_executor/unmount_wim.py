# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Format strings for use in unmount cmdline options
UNMOUNT_CMD = 'Dismount-WindowsImage'
UNMOUNT_DIR = '-Path "{}"'
UNMOUNT_LOG_PATH = '-LogPath "{}"'
UNMOUNT_LOG_LEVEL = '-LogLevel {}'
UNMOUNT_SAVE = '-Save'  # Save changes to the wim
UNMOUNT_DISCARD = '-Discard'  # Discard changes to the wim


def unmount_win_wim(powershell,
                    mnt_dir,
                    logs,
                    log_level='WarningsInfo',
                    save=True):
  """Unmount the winpe wim from the given directory"""
  # Args for the unmount cmd
  args = [
      UNMOUNT_DIR.format(mnt_dir),
      UNMOUNT_LOG_PATH.format(logs.join(UNMOUNT_CMD, 'unmount.log')),
      UNMOUNT_LOG_LEVEL.format(log_level)
  ]
  # Save/Discard the changes to the wim
  if save:
    args.append(UNMOUNT_SAVE)
  else:
    args.append(UNMOUNT_DISCARD)
  powershell(
      'Unmount wim at {}'.format(mnt_dir),
      UNMOUNT_CMD,
      logs=[logs.join(UNMOUNT_CMD)],
      args=args)
