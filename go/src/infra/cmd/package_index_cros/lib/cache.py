import json
import os
from typing import List

from .constants import CACHE_PACKAGES_PATH_TEMPALATE
from .logger import g_logger
from .package import Package
from .util import Setup


class PackageCache:
  """Responsible for caching packages."""

  def __init__(self, setup: Setup):
    self.setup = setup
    self.cache_filename = CACHE_PACKAGES_PATH_TEMPALATE.format(self.setup.board)

  def HasCachedPackages(self) -> bool:
    return os.path.isfile(self.cache_filename)

  def Clear(self) -> None:
    if self.HasCachedPackages():
      os.remove(self.cache_filename)

  def Store(self, packages: List[Package]) -> None:
    with open(self.cache_filename, 'w') as output:
      cache = {p.name: Package.Serialize(p) for p in packages}
      json.dump(cache, output)

  def Restore(self) -> List[Package]:
    assert self.HasCachedPackages(), 'Attempt to restore non-existing cache'

    with open(self.cache_filename) as input:
      cache = json.load(input)
      packages = []
      for p_name in cache:
        try:
          packages.append(Package.Deserialize(cache[p_name], self.setup))
        except Package.UnsupportedPackageException as e:
          g_logger.debug('%s: Skipping cached unsupported package: %s',
                         e.package_name, e.reason)
      return packages


class CacheProvider:
  """
  Responsible for providing an access to the actual caches.

  The caches can be null meaning there's no access or no cache.
  """

  def __init__(self, *, package_cache: PackageCache = None):
    self.package_cache = package_cache

  def Clear(self) -> None:
    """Clear all available caches."""

    if self.package_cache:
      self.package_cache.Clear()
