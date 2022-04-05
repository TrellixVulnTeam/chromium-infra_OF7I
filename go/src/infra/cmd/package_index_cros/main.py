#!/bin/python3

import argparse
import logging
import os
import sys
import textwrap

from lib.logger import g_logger
from lib.logger import SetupLogger


def _FindChromite(path):
  """Find the chromite dir in a repo, gclient, or submodule checkout."""
  path = os.path.abspath(path)
  # Depending on the checkout type (whether repo chromeos or gclient chrome)
  # Chromite lives in a different location.
  roots = (
      ('.repo', 'chromite/.git'),
      ('.gclient', 'src/third_party/chromite/.git'),
      ('src/.gitmodules', 'src/third_party/chromite/.git'),
  )

  while path != '/':
    for root, chromite_git_dir in roots:
      if all(
          os.path.exists(os.path.join(path, x))
          for x in [root, chromite_git_dir]):
        return os.path.dirname(os.path.join(path, chromite_git_dir))
    path = os.path.dirname(path)
  return None


def _MissingErrorOut(target):
  sys.stderr.write("""ERROR: Couldn't find the chromite tool %s.

Please change to a directory inside your Chromium OS source tree
and retry.  If you need to setup a Chromium OS source tree, see
  https://chromium.googlesource.com/chromiumos/docs/+/HEAD/developer_guide.md
""" % target)
  return 127


def _BuildParser():

  parser = argparse.ArgumentParser(
      formatter_class=argparse.RawTextHelpFormatter)

  parser.usage = '%(prog)s [options] package [package ...]'

  parser.description = textwrap.dedent("""\
    Generate compile commands and gn targets for given packages in current or
    given directory.""")

  parser.epilog = textwrap.dedent("""\
    If you don't want build artifcats, run: cros clean

    WARNING: Be careful with header files. There are still some include
    paths in chroot (like dbus, or standard library, or something else
    yet to be discovered). You might end up chanching a chroot file instead
    of the actual one.

    WARNING: --build-dir flag removes existing build dir if any.""")

  parser.add_argument(
      '--verbose',
      '-v',
      action='store_true',
      default=False,
      dest='verbose',
      help='Use DEBUG level for logging instead of default WARNING.')

  parser.add_argument('--with-build',
                      '--with_build',
                      action='store_true',
                      default=False,
                      dest='with_build',
                      help=textwrap.dedent("""\
    Build packages before generating.
    If you've already built packages
    and want to regenerate, you may
    skip this option."""))

  parser.add_argument(
      '--with-tests',
      '--with_tests',
      action='store_true',
      default=False,
      dest='with_tests',
      help=textwrap.dedent("""\
    Build tests alongside packages before generating.
    This assumes --with-build is set."""))

  parser.add_argument('--board',
                      '-b',
                      type=str,
                      default='amd64-generic',
                      dest='board',
                      help='Board to setup and build packages')

  parser.add_argument('--force',
                      '-f',
                      action='store_true',
                      default=False,
                      dest='force',
                      help='If set, clear cache')

  parser.add_argument(
      '--keep-going',
      '--keep_going',
      action='store_true',
      default=False,
      dest='keep_going',
      help="""\
    If set, skips failed packages and continues execution.""")

  parser.add_argument(
      '--skip-packages',
      '--skip_packages',
      type=str,
      default='',
      dest='skip_packages',
      help="""\
    String with space-separated list of full named
    packages to be ignored and skipped.""")

  compile_commands_args = parser.add_mutually_exclusive_group()
  compile_commands_args.add_argument('--compile-commands',
                                     '--compile_commands',
                                     '-c',
                                     type=str,
                                     dest='compile_commands_file',
                                     default=None,
                                     help=textwrap.dedent("""\
    Output file for compile commands json.
    Default: compile_commands.json in current directory.
    If --build-dir is specified, paths will refer to this
    directory."""))

  gn_targets_args = parser.add_mutually_exclusive_group()
  gn_targets_args.add_argument('--gn-targets',
                               '--gn_targets',
                               '-t',
                               type=str,
                               dest='gn_targets_file',
                               default=None,
                               help=textwrap.dedent("""\
    Output file for gn targets json.
    Default: gn_targets.json in current directory.
    If --build-dir is specified, paths will refer to this
    directory."""))

  build_dir_args = parser.add_mutually_exclusive_group()
  build_dir_args.add_argument('--build-dir',
                              '--build_dir',
                              '-o',
                              type=str,
                              dest='build_dir',
                              default=None,
                              help=textwrap.dedent("""\
    WARNING: existing dir if any will be completely
    removed.

    Directory to store build artifacts from out/Default
    packages dirs.
    If --build-dir is specified, paths will refer to this
    directory."""))

  parser.add_argument('packages',
                      type=str,
                      nargs='+',
                      help='List of packages to generate')

  return parser


def main():
  parser = _BuildParser()
  args = parser.parse_args()
  if args.compile_commands_file:
    args.compile_commands_file = os.path.abspath(args.compile_commands_file)

  if args.gn_targets_file:
    args.gn_targets_file = os.path.abspath(args.gn_targets_file)

  if args.build_dir:
    args.build_dir = os.path.abspath(args.build_dir)

  SetupLogger(logging.DEBUG if args.verbose else logging.WARNING)

  chromite_dir = _FindChromite(os.getcwd())
  if not chromite_dir:
    return _MissingErrorOut(sys.argv[0])
  g_logger.debug('Chromite dir: %s', chromite_dir)
  sys.path.append(os.path.dirname(chromite_dir))

  from lib.cache import CacheProvider
  from lib.cache import PackageCache
  from lib.conductor import Conductor
  from lib.util import Setup

  setup = Setup(
      args.board,
      skip_packages=args.skip_packages.split(' '),
      with_tests=args.with_tests)
  cache_provider = CacheProvider(package_cache=PackageCache(setup))

  if args.force:
    cache_provider.Clear()

  cache_provider.package_cache = None
  conductor = Conductor(setup=setup, cache_provider=cache_provider)
  conductor.Prepare(package_names=args.packages, with_build=args.with_build)
  conductor.DoMagic(
      cdb_output_file=args.compile_commands_file,
      targets_output_file=args.gn_targets_file,
      build_output_dir=args.build_dir,
      keep_going=args.keep_going)


if __name__ == '__main__':
  main()
