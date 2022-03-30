from typing import List
import os

from .build_dir import BuildDirGenerator
from .cache import CacheProvider
from .cdb import CdbGenerator
from .gn_targets import GnTargetsGenerator
from .logger import g_logger
from .package import Package
from .package_sleuth import PackageSleuth
from .util import CrosSdk
from .util import Setup


class Conductor:
  """Ochestrates whole process."""

  def __init__(self, setup: Setup, cache_provider: CacheProvider):
    self.setup = setup
    self.cros_sdk = CrosSdk(self.setup)
    self.cache_provider = cache_provider

  def Prepare(self, package_names: List[str], *, with_build: bool = False):
    """
    Does:
      * List packages:
        * If |package_names| - fetches given packages and their dependencies.
        * If not |package_names| - fetches all available packages.
      * If |with_build|:
        * Build packages.
    """

    assert os.path.isdir(
        self.setup.board_dir), f"Board is not set up: {self.setup.board}"

    package_sleuth = PackageSleuth(self.setup,
                                   cache=self.cache_provider.package_cache)
    packages_list, ignored_packages_list = package_sleuth.ListPackages(
        packages_names=package_names)

    assert packages_list, 'No packages to work with'
    assert len(packages_list) == len(set([p.name for p in packages_list
                                         ])), 'Duplicates among packages'

    if ignored_packages_list:
      g_logger.warning('Following packages are not supported and ignored: %s',
                       ignored_packages_list)

    if self.cache_provider.package_cache:
      self.cache_provider.package_cache.Store(packages_list)

    # Sort packages so that dependencies go first.
    self.packages = Conductor._GetSortedPackages(packages_list)

    if with_build:
      package_names = [p.name for p in self.packages]
      with self.cros_sdk.StartWorkonPackagesSafe(
          package_names) as workon_handler:
        self.cros_sdk.BuildPackages(package_names)

  def DoMagic(self,
              *,
              cdb_output_file: str = None,
              targets_output_file: str = None,
              build_output_dir: str = None,
              keep_going: bool = False):
    """
    Calls generators one by one. |Prepare| should be called prior this method.
    """
    for p in self.packages:
      p.Initialize()

    build_dir_conflicts = {}
    if build_output_dir:
      build_dir_conflicts = BuildDirGenerator(self.setup).Generate(
          self.packages, build_output_dir)
      g_logger.info('Generated build dir: %s', build_output_dir)

    if cdb_output_file:
      CdbGenerator(
          self.setup,
          result_build_dir=build_output_dir,
          file_conflicts=build_dir_conflicts,
          keep_going=keep_going).Generate(self.packages, cdb_output_file)
      g_logger.info('Generated cdb file: %s', cdb_output_file)

    if targets_output_file:
      GnTargetsGenerator(
          self.setup,
          result_build_dir=build_output_dir,
          file_conflicts=build_dir_conflicts,
          keep_going=keep_going).Generate(self.packages, targets_output_file)
      g_logger.info('Generated targets file: %s', targets_output_file)

    g_logger.info('Done')

  @staticmethod
  def _GetSortedPackages(packages_list: List[Package]) -> List[Package]:
    """
    Returns topologically sorted packages.

    Packages graph is according to package dependencies: more independent
    packages go first.
    """
    result_packages = []
    packages_dict = {p.name: p for p in packages_list}

    in_degrees = {p.name: 0 for p in packages_list}
    for p in packages_list:
      for dep in p.dependencies:
        in_degrees[dep.name] = in_degrees[dep.name] + 1

    queue = [p_name for p_name in in_degrees if in_degrees[p_name] == 0]
    while queue:
      p_name = queue.pop(0)
      result_packages.append(packages_dict[p_name])
      for dep in packages_dict[p_name].dependencies:
        in_degrees[dep.name] = in_degrees[dep.name] - 1
        if in_degrees[dep.name] == 0:
          queue.append(dep.name)
      assert len(result_packages) <= len(
          packages_list
      ), 'Too many sorted packages. Probably because of circular dependencies'

    assert len(result_packages) == len(packages_list), "Missing some packages"

    result_packages.reverse()
    return result_packages
