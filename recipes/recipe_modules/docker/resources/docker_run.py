# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from __future__ import print_function

import argparse
import os
import platform
import shutil
import subprocess
import sys
import tempfile


def main():
  parser = argparse.ArgumentParser()
  parser.add_argument(
      '--image', required=True, help='Image in which command will be executed.')
  parser.add_argument(
      '--config-file',
      required=True,
      help='Location of the Docker config file.')
  parser.add_argument(
      '--dir-map',
      metavar=('HOST_DIR', 'DOCKER_PATH'),
      nargs=2,
      action='append',
      default=[],
      help='Directories to be mapped in host_path:docker_path. Host paths '
      'that do not exist will be created before running docker to make '
      'sure that they are owned by current user.')
  parser.add_argument(
      '--env',
      action='append',
      default=[],
      help='Environment variable strings, e.g. foo=bar')
  parser.add_argument(
      '--inherit-luci-context',
      action='store_true',
      default=False,
      help='Inherit current LUCI Context (including auth). '
      'CAUTION: removes network isolation between the container and the '
      'docker host. Read more https://docs.docker.com/network/host/')

  args, command = parser.parse_known_args()
  if command and command[0] == '--':
    command = command[1:]

  cmd = [
      'docker',
      '--config',
      args.config_file,
      'run',
      '--user',
      '%s:%s' % (os.getuid(), os.getgid()),
  ]

  for host_path, docker_path in args.dir_map:
    # Ensure that host paths exist, otherwise they will be created by the docker
    # command, which makes them owned by root and thus hard to remove/modify.
    if not os.path.exists(host_path):
      os.makedirs(host_path)
    elif not os.path.isdir(host_path):
      parser.error('Cannot map non-directory host path: %s' % host_path)
    cmd.extend(['--volume', '%s:%s' % (host_path, docker_path)])

  for var in args.env:
    cmd.extend(['--env', var])

  new_ctx_file = None
  if args.inherit_luci_context:
    if platform.system() != 'Linux':
      print(
          '--inherit-luci-context is supported only on Linux', file=sys.stderr)
      return 1

    cur_ctx_file = os.environ.get('LUCI_CONTEXT')
    if not cur_ctx_file:
      print('$LUCI_CONTEXT is not set or empty', file=sys.stderr)
      return 1

    # Copy the context file to make it available to Docker service.
    new_ctx_file = tempfile.mktemp(prefix='luci_context')
    shutil.copyfile(cur_ctx_file, new_ctx_file)

    cmd.extend([
        # Map the temp file to /tmp/luci_context inside the container.
        '--volume',
        '%s:/tmp/luci_context' % new_ctx_file,
        # Add LUCI_CONTEXT variable pointing to /tmp/luci_context inside the
        # container.
        '--env',
        'LUCI_CONTEXT=/tmp/luci_context',
        # Remove network isolation, s.t. the container can contact local LUCI
        # auth server.
        '--network',
        'host',
    ])

  cmd.append(args.image)
  cmd.extend(command)
  try:
    subprocess.check_call(cmd)
  except subprocess.CalledProcessError as e:
    return e.returncode
  finally:
    if new_ctx_file:
      os.remove(new_ctx_file)


if __name__ == '__main__':
  sys.exit(main())
