import os
from typing import List

from chromite.lib import cros_build_lib
from chromite.lib import git
from chromite.lib import path_util
from chromite.lib import repo_util

from .constants import PRINT_DEPS_SCRIPT_PATH
from .logger import g_logger


class Setup:
  """
  POD to keep all data related to a setup:
    * board
    * cros_dir: absolute path to chromeos checkout root dir
    * chroot_dir: absolute path to chroot dir
    * src_dir: absolute path to src
    * manifest: manifest handler
  """

  def __init__(self,
               board: str,
               *,
               skip_packages: List[str] = None,
               with_tests: bool = False):
    self.board = board

    checkout_info = path_util.DetermineCheckout()
    if checkout_info.type != path_util.CHECKOUT_TYPE_REPO:
      raise repo_util.NotInRepoError(
          'Script is executed outside of ChromeOS checkout')

    self.cros_dir = checkout_info.root
    self.chroot_dir = path_util.FromChrootPath('/', self.cros_dir)
    self.board_dir = os.path.join(self.chroot_dir, 'build', self.board)
    self.src_dir = os.path.join(self.cros_dir, 'src')
    self.platform2_dir = os.path.join(self.src_dir, 'platform2')

    # List of dirs that might not exist and can be ignored during path fix.
    self.ignorable_dirs = [
        os.path.join(self.board_dir, 'usr', 'include', 'chromeos', 'libsoda'),
        os.path.join(self.board_dir, 'usr', 'share', 'dbus-1'),
        os.path.join(self.board_dir, 'usr', 'share', 'proto'),
        os.path.join(self.chroot_dir, 'usr', 'include', 'android'),
        os.path.join(self.chroot_dir, 'usr', 'include', 'cros-camera'),
        os.path.join(self.chroot_dir, 'usr', 'lib64', 'shill'),
        os.path.join(self.chroot_dir, 'usr', 'libexec', 'ipsec'),
        os.path.join(self.chroot_dir, 'usr', 'libexec', 'l2tpipsec_vpn'),
        os.path.join(self.chroot_dir, 'usr', 'share', 'cros-camera'),
        os.path.join(self.chroot_dir, 'build', 'share'),
    ]

    self.skip_packages = skip_packages
    self.with_tests = with_tests

  @property
  def manifest(self) -> git.ManifestCheckout:
    return git.ManifestCheckout.Cached(self.cros_dir)


class CrosSdk:
  """Handles requests to cros_sdk."""

  @staticmethod
  def _Exec(cmd, *, capture_output=False) -> cros_build_lib.CommandResult:
    shell = True
    if isinstance(cmd, List):
      shell = False

    encoding = 'utf-8' if capture_output else None

    g_logger.debug("Executing: '%s'", cmd)
    res = cros_build_lib.run(cmd,
                             enter_chroot=True,
                             shell=shell,
                             capture_output=capture_output,
                             encoding=encoding,
                             check=True,
                             print_cmd=False)
    return res

  def __init__(self, setup: Setup):
    self.setup = setup

  def StartWorkonPackages(self, package_names: List[str]) -> None:
    """
    Runs cros_workon start with given packages.

    Raises:
      * cros_build_lib.CommandResult if command failed.
    """
    cmd = ['cros_workon', f"--board={self.setup.board}", "start"
          ] + package_names
    CrosSdk._Exec(cmd)

  def StopWorkonPackages(self, package_names: List[str]) -> None:
    """
    Runs cros_workon stop with given packages.

    Raises:
      * cros_build_lib.CommandResult if command failed.
    """
    cmd = ['cros_workon', f"--board={self.setup.board}", "stop"] + package_names
    CrosSdk._Exec(cmd)

  def StartWorkonPackagesSafe(self, package_names: List[str]):
    """
    Safe version of start/stop workon. Preserves state of already workon
    packages.

    Returns a handler to stop workon with 'with' statement.

    Raises:
      * cros_build_lib.CommandResult if command failed.
    """
    check_workon_cmd = ['cros_workon', f"--board={self.setup.board}", "list"]

    output = CrosSdk._Exec(check_workon_cmd, capture_output=True).output
    workon_packages = set(output.split('\n'))
    non_workon_packages = [p for p in package_names if not p in workon_packages]

    g_logger.debug('Packages to start workon: %s', non_workon_packages)
    g_logger.debug('Packages already workon: %s',
                   [p for p in package_names if p in workon_packages])

    class WorkonRaii:

      def __init__(self, cros_sdk: CrosSdk, packages_to_workon: List[str]):
        self.cros_sdk = cros_sdk
        self.packages_to_workon = packages_to_workon

      def __enter__(self) -> 'WorkonRaii':
        if self.packages_to_workon and len(self.packages_to_workon) != 0:
          self.cros_sdk.StartWorkonPackages(self.packages_to_workon)
        return self

      def __exit__(self, type, value, traceback) -> None:
        if self.packages_to_workon and len(self.packages_to_workon) != 0:
          self.cros_sdk.StopWorkonPackages(self.packages_to_workon)

    return WorkonRaii(self, non_workon_packages)

  def BuildPackages(self, package_names: List[str]) -> None:
    """
    Builds given packages and preserves build artifcats.

    Raises:
      * cros_build_lib.CommandResult if command failed.
    """
    features = ['noclean']
    if self.setup.with_tests:
      features.append('test')
    cmd = ' '.join([
        'sudo', f'FEATURES="{" ".join(features)}"', 'parallel_emerge',
        '--board', self.setup.board
    ] + package_names)
    CrosSdk._Exec(cmd)

  def GenerateCompileCommands(self, chroot_build_dir: str) -> str:
    """
    Calls ninja and returns compile commands as a string.

    |chroot_build_dir|  is package's build dir inside chroot.

    Raises:
      * cros_build_lib.CommandResult if command failed.
    """
    ninja_cmd = ['ninja', '-C', chroot_build_dir, '-t', 'compdb', 'cc', 'cxx']

    return CrosSdk._Exec(ninja_cmd, capture_output=True).output

  def GenerateGnTargets(self, chroot_root_dir: str,
                        chroot_build_dir: str) -> str:
    """
    Calls gn desc and returns gn targets as a string.

    |chroot_root_dir| is a package's dir containing upper most .gn file inside
    chroot.
    |chroot_build_dir| is a package's build dir inside chroot.

    Raises:
      * cros_build_lib.CommandResult if command failed.
    """
    gn_desc_cmd = [
        'gn',
        'desc',
        f"--root={chroot_root_dir}",
        chroot_build_dir,
        '*',
        '--format=json',
    ]

    return CrosSdk._Exec(gn_desc_cmd, capture_output=True).output

  def GenerateDependencyTree(self, package_names: List[str]):
    """
    Generates dependency tree for given packages.

    Returns a dictionary with dependencies (see script/print_deps.py for
    detailed format).

    Utilizes chromite.lib.depgraph to fetch dependency tree. Depgraph has to be
    called from inside chroot so it lives in separate script file which is
    called via cros_sdk wrapper.

    Raises:
      * cros_build_lib.CommandResult if command failed.
    """
    cmd = [
        'sudo',
        path_util.ToChrootPath(PRINT_DEPS_SCRIPT_PATH), self.setup.board
    ] + package_names
    return CrosSdk._Exec(cmd, capture_output=True).output
