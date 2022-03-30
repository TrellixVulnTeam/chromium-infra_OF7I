import filecmp
import os
import shutil
from typing import Dict, List

from .logger import g_logger
from .package import Package
from .util import Setup


class _BuildDirMerger:
  """Merges build directories of given packages."""

  g_ignore_extensions = [
      '.gn',
      '.ninja',
      '.ninja.d',
      '.ninja_deps',
      '.ninja_log',
  ]

  def __init__(self, setup: Setup, result_build_dir):
    self.setup = setup
    self.result_build_dir = result_build_dir

    assert os.path.isdir(
        self.result_build_dir), 'Result build dir does not exist'

  def Append(self, package: Package) -> Dict:
    """
    Add |package|'s build dir to result one.

    Returns a dictionary of conflicting files (same result name, different
    content) mapping file's original name to a result name. The result name
    is composed like {dest_dir}/{package_name}_{filename}.
    """
    source_dest_conflicts = {}

    def CopyFile(source: str, dest: str) -> None:
      assert os.path.isfile(source), 'Copying directory instead of file'

      if any([
          source.endswith(ext) for ext in _BuildDirMerger.g_ignore_extensions
      ]):
        g_logger.debug('%s: ignore file: %s', package.name, source)
        return

      if os.path.exists(dest) and not filecmp.cmp(source, dest):
        dest = os.path.join(os.path.dirname(dest),
                            package.simple_name + '_' + os.path.basename(dest))
        g_logger.debug(
            '%s: Copying conflicting file with package prefix: %s to %s',
            package.name, source, dest)
        source_dest_conflicts[source] = dest
      shutil.copy2(source, dest)

    def CopyDir(source: str, dest: str) -> None:
      assert os.path.isdir(source), 'Copying file instead of directory'

      for item in os.listdir(source):
        source_item = os.path.join(source, item)
        dest_item = os.path.join(dest, item)

        if os.path.isdir(source_item):
          if not os.path.isdir(dest_item):
            os.mkdir(dest_item)
          CopyDir(source_item, dest_item)
        else:
          CopyFile(source_item, dest_item)

    CopyDir(package.build_dir, self.result_build_dir)
    return source_dest_conflicts


class BuildDirGenerator:
  """Merges build directories of given packages."""

  def __init__(self, setup: Setup):
    self.setup = setup

  def _PrepareDir(self, result_build_dir: str) -> None:
    """Delete existing |result_build_dir| if exist. Create new one."""

    if os.path.isdir(result_build_dir):
      g_logger.warning('Removing existing build dir: %s', result_build_dir)
      shutil.rmtree(result_build_dir)

    os.makedirs(result_build_dir)
    g_logger.debug('Build dir created: %s', result_build_dir)

  def Generate(self, packages: List[Package], result_build_dir: str) -> Dict:
    """
    Generates common |result_build_dir| accumulating artifcats from |packages|'s
    build dirs.

    Returns a dictionary of conflicting files (same result name, different
    content) mapping file's original name to a result name. The result name is
    composed like {dest_dir}/{package_name}_{filename}.
    """
    assert result_build_dir

    self._PrepareDir(result_build_dir)

    merger = _BuildDirMerger(self.setup, result_build_dir)
    source_dest_conflicts = {}
    for package in packages:
      source_dest_conflicts.update(merger.Append(package))
      g_logger.debug('Added %s to result build dir: %s', package.name,
                     package.build_dir)

    return source_dest_conflicts
