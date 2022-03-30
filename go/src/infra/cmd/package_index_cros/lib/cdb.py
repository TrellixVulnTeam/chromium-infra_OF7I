import filecmp
import json
import os
from typing import Dict, NamedTuple, List, Set, Tuple

from .logger import g_logger
from .package import PackagePathException
from .package import Package
from .path_handler import PathHandler
from .util import CrosSdk
from .util import Setup


class Cdb:
  """Responsible for fixing paths in compile commands database."""

  class CdbException(Exception):
    """Exception to indicate failure while fixing Cdb."""

  class DirectoryFieldException(CdbException, PackagePathException):
    """Exception to indicate failure while resolving directory field"""

  class FileFieldException(CdbException, PackagePathException):
    """Exception to indicate failure while resolving file field"""

  class ArgumentsFieldExcpetion(CdbException, PackagePathException):
    """Exception to indicate a failure while resolving command or arguments field"""

  class OutputFieldException(CdbException, PackagePathException):
    """Exception to indicate failure while resolving output field"""

  class _IncludePathOrder(NamedTuple):
    """
    Sort include args by interest:
    * local from {cros_dir}/src,
    * generated in {build_dir}
    * TODO: chroot paths shall be skipped in favor of include paths from
      dependencies.
    """

    local: Set[str]
    generated: Set[str]
    chroot: Set[str]

  g_clang_additional_args = ['-stdlib=libc++']

  def __init__(self,
               cdb_data: List,
               package: Package,
               setup: Setup,
               package_to_include_args: Dict[str, 'Cdb._IncludePathOrder'],
               *,
               result_build_dir: str = None,
               file_conflicts: Dict = {}):
    """
    Cdb constructor.

    Arguments:
      * cdb_data: loaded of compile_commands.json for |package|.
      * package: package to work with.
      * setup: setup data (board, dirs, etc).
      * package_to_include_args: maps packages to their include dirs. Is used to
        populate |package| dependencies' include paths.
      * result_build_dir: path to result build dir simulating single result
        package.
      * file_conflicts: maps original build artifacts in chroot dir that
        conflict between packages to corresponding artfacts in
        |result_build_dir|.
    """
    self.data = cdb_data
    self.package = package
    self.setup = setup
    self.path_handler = PathHandler(self.setup)
    if result_build_dir:
      self.build_dir = result_build_dir
    else:
      self.build_dir = self.package.build_dir
    self.file_conflicts = file_conflicts

    for dep in self.package.dependencies:
      if dep.name not in package_to_include_args:
        raise Cdb.CdbException(
            f"{self.package.name}: No include path for dependency: {dep.name}")

    self.package_to_include_args = package_to_include_args
    self.package_to_include_args[self.package.name] = Cdb._IncludePathOrder(
        set(), set(), set())

  def Fix(self) -> 'Cdb':
    """
    Fix cdb entries:
    * Substitute chroot paths with corresponding paths outside of chroot.
    * Substitute temp src paths with actual paths.
    * TODO: substitute chroot include paths with actual paths from
      dependencies.
    * Add several clang args.
    """
    if self.package.is_highly_volatile:
      g_logger.debug('%s: Is highly volatile package. Not all checks performed',
                     self.package.name)

    if self.package.additional_include_paths:
      for include_path in self.package.additional_include_paths:
        g_logger.debug('%s: Additional include path will be used: %s',
                       self.package.name, include_path)

    for entry in self.data:
      entry['directory'] = self._GetFixedDirectory(entry)

      entry['file'] = os.path.relpath(self._GetFixedFile(entry),
                                      entry['directory'])

      entry['command'] = ' '.join(self._GetFixedArguments(entry))
      if 'arguments' in entry:
        del entry['arguments']

      if 'output' in entry:
        entry['output'] = self._GetFixOutput(entry)

    return self

  def _GetFixedDirectory(self, entry: Dict) -> str:
    assert 'directory' in entry, 'Directory field is missing'

    dir = self.path_handler.FromChroot(entry['directory'])

    if dir != self.package.build_dir:
      raise Cdb.DirectoryFieldException(
          self.package, 'Directory field does not match build dir', dir,
          self.package.build_dir)

    return self.build_dir

  def _GetFixedArguments(self, entry: Dict) -> List[str]:
    # Each entry has either command or arguments. If it's arguments then
    # substitute it with command.
    assert 'arguments' in entry or 'command' in entry, \
       'Arguments and command field are missing'

    if 'arguments' in entry:
      arguments = entry['arguments']
    else:
      arguments = entry['command'].split(' ')

    # First argument is always a compiler.
    actual_arguments = [self._FixArgumentsCompiler(arguments[0])]
    actual_include_args = Cdb._IncludePathOrder(set(), set(), set())

    for arg in arguments:

      def Fixer(chroot_path):
        return self._FixPath(chroot_path,
                             ignore_generated=True,
                             ignore_highly_volatile=True,
                             ignorable_dirs=self.setup.ignorable_dirs).actual

      if arg[0:2] == '-I':
        # Include path arg is more strict to fix.
        actual_path = self._FixPath(arg[2:], ignore_generated=True).actual
        actual_arg = '-I' + actual_path
        if actual_path.startswith(self.build_dir):
          # If actual_include_path.startswith(self.build_dir):
          # Build dir can be inside {src_dir}. So it comes before local.
          actual_include_args.generated.add(actual_arg)
        elif actual_path.startswith(self.setup.src_dir):
          actual_include_args.local.add(actual_arg)
        elif actual_path.startswith(self.setup.chroot_dir):
          actual_include_args.chroot.add(actual_arg)
        else:
          raise NotImplementedError(f"Unexpected include path: {actual_path}")
      else:
        arg_prefix, actual_path = PathHandler.FixPathInArgument(arg, Fixer)
        actual_arg = arg_prefix + actual_path
        actual_arguments.append(actual_arg)

    # Args are fixed.

    if self.package.additional_include_paths:
      for include_path in self.package.additional_include_paths:
        actual_include_args.local.add('-I' + include_path)

    # Do not pass our dependencies up.
    self.package_to_include_args[self.package.name].local.update(
        actual_include_args.local)
    self.package_to_include_args[self.package.name].generated.update(
        actual_include_args.generated)

    for dep in self.package.dependencies:
      actual_include_args.local.update(
          self.package_to_include_args[dep.name].local)
      actual_include_args.generated.update(
          self.package_to_include_args[dep.name].generated)

    actual_arguments.extend(Cdb.g_clang_additional_args)
    actual_arguments.extend(actual_include_args.local)
    actual_arguments.extend(actual_include_args.generated)
    actual_arguments.extend(actual_include_args.chroot)

    return actual_arguments

  def _GetFixedFile(self, entry: Dict) -> str:
    assert 'file' in entry, 'File field is missing'

    temp_file, actual_file = self._FixPath(entry['file'],
                                           ignore_generated=True,
                                           ignore_highly_volatile=True)

    if temp_file != actual_file:
      if not os.path.isfile(temp_file) or not os.path.isfile(actual_file):
        g_logger.debug(
            '%s: Cannot verify if temp and actual file are the same: %s vs %s',
            self.package.name, temp_file, actual_file)
      elif not filecmp.cmp(temp_file, actual_file):
        if self.package.is_highly_volatile:
          g_logger.debug(
              '%s: Temp and actual files differ. Possibly patches: %s vs %s',
              self.package.name, temp_file, actual_file)
        else:
          raise Cdb.FileFieldException(self.package,
                                       'Temp and actual file differ', temp_file,
                                       actual_file)

    return actual_file

  def _GetFixOutput(self, entry: Dict) -> str:
    assert 'output' in entry, 'Output field is missing'

    actual_file = self._FixPath(entry['output'],
                                ignore_generated=True,
                                ignore_highly_volatile=True).actual

    return actual_file

  def _FixPath(self, chroot_path: str, **ignore_args) -> PathHandler.FixedPath:
    """
    Wrapper for |PathHandler.FixPathWithIgnores| with additional action to
    move path from |package.build_dir| to |self.result_build_dir|.
    """
    fixed_path = self.path_handler.FixPathWithIgnores(
        chroot_path,
        self.package,
        conflicting_paths=self.file_conflicts,
        **ignore_args)

    if fixed_path.actual.startswith(self.package.build_dir):
      return PathHandler.FixedPath(
          fixed_path.original,
          PathHandler.MovePath(fixed_path.actual, self.package.build_dir,
                               self.build_dir))
    return fixed_path

  def _FixArgumentsCompiler(self, compiler: str) -> str:
    if compiler.endswith('clang++'):
      return 'clang++'
    elif compiler.endswith('clang'):
      return 'clang'
    else:
      raise NotImplementedError(f"Unknown compiler: '{compiler}'")


class CdbGenerator:
  """Generates, fixes and merges compile databases for given packages."""

  def __init__(self,
               setup: Setup,
               *,
               result_build_dir: str = None,
               file_conflicts: Dict = {},
               keep_going: bool = False):
    """
    CdbGenerator constructor.

    Arguments:
      * setup: setup data (board, dirs, etc).
      * result_build_dir: path to result build dir simulating single result
        package.
      * file_conflicts: maps original build artifacts in chroot dir that
        conflict between packages to corresponding artfacts in
        |result_build_dir|.
    """
    self.setup = setup
    self.result_build_dir = result_build_dir
    self.file_conflicts = file_conflicts
    self.keep_going = keep_going

  def _GenerateCdbForPackage(self, package: Package,
                             packages_to_include_args: Dict) -> Cdb:
    cdb_str = CrosSdk(self.setup).GenerateCompileCommands(
        PathHandler(self.setup).ToChroot(package.build_dir))
    g_logger.debug('%s: Generated compile commands', package.name)

    cdb_data = json.loads(cdb_str)
    if not cdb_data:
      g_logger.error('%s: Compile commands are empty', package.name)

    assert isinstance(cdb_data, List)

    return Cdb(cdb_data,
               package,
               self.setup,
               packages_to_include_args,
               result_build_dir=self.result_build_dir,
               file_conflicts=self.file_conflicts)

  def _GenerateResultCdb(self, packages: List[Package]) -> List:
    result_cdb_data = []

    packages_to_include_args = {}
    for package in packages:
      try:
        cdb_data = self._GenerateCdbForPackage(
            package, packages_to_include_args).Fix().data
        result_cdb_data.extend(cdb_data)
      except (Cdb.CdbException, PackagePathException) as e:
        if self.keep_going:
          g_logger.error('%s: Failed to fix compile commands: %s', package.name,
                         e)
        else:
          raise e

    return result_cdb_data

  def Generate(self, packages: List[Package], result_cdb_file: str) -> None:
    """
    Generates, fixes and merges compile databases for given |packages|.

    Raises:
      * Cdb.CdbException or field specific exception if failed to fix cdb entry.
    """
    assert result_cdb_file

    result_cdb = self._GenerateResultCdb(packages)

    with open(result_cdb_file, 'w') as output:
      json.dump(result_cdb, output, indent=2)
