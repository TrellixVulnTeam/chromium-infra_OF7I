import json
from typing import Dict, List, NamedTuple, Set

from chromite.lib import portage_util

import lib.package as pkg
from .cache import PackageCache
from .logger import g_logger
from .util import Setup
from .util import CrosSdk


class PackageSleuth:
  """Handles requests to find packages."""

  class SupportedUnsupportedPackages(NamedTuple):
    supported: List[pkg.Package]
    unsupported: List[str]

  def __init__(self, setup: Setup, *, cache: PackageCache = None):
    self.setup = setup
    self.cache = cache
    self.overlays = portage_util.FindOverlays(
        overlay_type=portage_util.constants.BOTH_OVERLAYS,
        board=self.setup.board,
        buildroot=self.setup.cros_dir)

  def ListPackages(self,
                   *,
                   packages_names: List[str] = []
                  ) -> SupportedUnsupportedPackages:
    """
    Returns list of packages for given |packages_names| or all available
    packages if |packages_names| is none or empty.

    Returns cached packages if any and |packages_names| is empty.

    Returns list of found unsupported packages as well.
    """
    packages_names = packages_names if packages_names is not None else []

    packages = self._ListPackagesWithCache(packages_names)
    PackageSleuth._FilterPackagesDependencies(packages.supported)

    return packages

  def _ListPackagesWithCache(
      self, packages_names: List[str]) -> SupportedUnsupportedPackages:
    """
    Returns fetched packages if there is no cache or |packages_names| not empty.

    Returns cached packages only if there are any and |packages_names| is empty.
    """

    if self.cache and self.cache.HasCachedPackages():
      g_logger.debug('Restoring packages from cache')
      cached_packages = self.cache.Restore()
      g_logger.debug('Number of packages restored from cache: %s',
                     len(cached_packages))
    else:
      cached_packages = []

    if not packages_names and cached_packages:
      return PackageSleuth.SupportedUnsupportedPackages(cached_packages, [])

    # Skip any other usage of cache altogether. No idea how to apply it yet.

    packages = self._ListPackagesWithDeps(packages_names)

    return packages

  def _ListPackagesWithDeps(
      self, packages_names: List[str]) -> SupportedUnsupportedPackages:
    """
    Returns list of packages for given |packages_names| extended with
    dependencies and dependencies' dependencies and more until the end.
    """

    packages = PackageSleuth.SupportedUnsupportedPackages([], [])

    ebuilds = self._ListEbuilds(packages_names)
    dependencies = self._GetPackagesDependencies([e.package for e in ebuilds])

    listed_packages = set([e.package for e in ebuilds])
    # List of packages names to list taken from current dependencies that don't
    # have corresponding ebuild yet.
    packages_to_list = [
        package_name for package_name in dependencies
        if package_name not in listed_packages
    ]

    # Repeating while we have packages without ebuilds and newly found
    # dependencies without corresponding ebuild.
    while packages_to_list:
      # It's not necessary that |new_ebuilds| == |packages_to_list|. new_ebuilds
      #  can be less or even empty.
      new_ebuilds = self._ListEbuilds(packages_to_list)
      if not new_ebuilds:
        break

      # TODO: Some packages need specifc USE flag for emerge (e.g. arc-base
      # needs USE=arcpp or USE=arcvm). Without them emerge fals and cros_sdk
      # raises an exception.
      new_dependencies = self._GetPackagesDependencies(
          [e.package for e in new_ebuilds])

      ebuilds += new_ebuilds
      dependencies.update(new_dependencies)
      listed_packages.update([e.package for e in new_ebuilds])

      packages_to_list = [
          package_name for package_name in new_dependencies
          if package_name not in listed_packages
      ]

    for ebuild in ebuilds:
      is_supported = pkg.IsPackageSupported(ebuild, self.setup)
      if is_supported != pkg.PackageSupport.SUPPORTED:
        g_logger.warning('%s: Not supported: %s', ebuild.package, is_supported)
        packages.unsupported.append(ebuild.package)
      else:
        packages.supported.append(
            pkg.Package(self.setup, ebuild, dependencies[ebuild.package]))

    return packages

  def _ListEbuilds(self, packages_names: List[str]) -> portage_util.EBuild:
    """
    Returns list of ebuilds for given |packages_names| or all available
    ebuilds if |packages_names| is none or empty.

    The number of returned ebuilds may be less than the number of
    |packages_names|. E.g. there can be a miss if requested package is private
    and we fetching only public packages. Or if requested package is out of
    scope for given board.
    """

    looking_for_all_packages = not packages_names
    ebuilds = []
    for o in self.overlays:
      ebuilds += portage_util.GetOverlayEBuilds(
          o, use_all=looking_for_all_packages, packages=packages_names)

    return ebuilds

  def _GetPackagesDependencies(
      self,
      packages_names: List[str]) -> Dict[str, List[pkg.PackageDependency]]:
    """
    Returns a dictionary mapping packages names to their dependencies.

    The dictionary size is greater than given |packages_names|. Dependencies are
    also mapped with depth = 1.
    """

    return self._GetPackagesDependenciesDepgraph(packages_names)

  def _GetPackagesDependenciesDepgraph(
      self,
      packages_names: List[str]) -> Dict[str, List[pkg.PackageDependency]]:
    """
    Returns a dictionary mapping packages names to their dependencies.

    The dictionary size is greater than given |packages_names|. Dependencies are
    also mapped with depth = 1.
    """

    deps_json = CrosSdk(self.setup).GenerateDependencyTree(packages_names)
    deps_tree = json.loads(deps_json)

    package_to_deps = {}
    for package in deps_tree:
      deps = deps_tree[package]['deps']
      package_name = PackageSleuth._TrimPackageVersion(package)
      package_to_deps[package_name] = [
          pkg.PackageDependency(PackageSleuth._TrimPackageVersion(d),
                                deps[d]['deptypes']) for d in deps
      ]

    # Check that all given packages have their deps fetched.
    assert not [
        package_name for package_name in packages_names
        if package_name not in package_to_deps
    ]

    return package_to_deps

  @staticmethod
  def _FilterPackagesDependencies(packages: List[pkg.Package]) -> None:
    supported_packages_names = set([p.name for p in packages])
    for package in packages:
      package.dependencies = PackageSleuth._GetFilterDependencies(
          package, supported_packages_names)

  @staticmethod
  def _GetFilterDependencies(
      package: pkg.Package,
      available_packages_names: Set[str]) -> List[pkg.PackageDependency]:

    def IsSupportedDependency(dep: pkg.PackageDependency) -> bool:
      # Filter package itself.
      if dep.name == package.name:
        return False

      # Filter unsupported or not queried dependencies.
      if dep.name not in available_packages_names:
        return False

      # Filter circular dependencies caused by PDEPEND.
      if len(dep.types) == 1 and 'runtime_post' in dep.types:
        return False

      return True

    return [dep for dep in package.dependencies if IsSupportedDependency(dep)]

  @staticmethod
  def _TrimPackageVersion(package_name: str) -> str:
    last_dash_pos = package_name.rfind('-')

    if last_dash_pos == -1:
      # No dashes. Assuming pure package name.
      return package_name

    if package_name[last_dash_pos + 1] != 'r':
      # Only version after dash. Trimming it.
      assert package_name[last_dash_pos + 1].isdigit()
      return package_name[:last_dash_pos]

    # Resping after dash. Trimming it as well.
    return PackageSleuth._TrimPackageVersion(package_name[:last_dash_pos])
