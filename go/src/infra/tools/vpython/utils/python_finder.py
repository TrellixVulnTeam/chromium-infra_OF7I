# pylint: skip-file
import argparse
import gopo
import json
import logging
import modulefinder
import os
import pprint
import sys
import tokenizer
import utilities
import urllib.request
import wheels

# CONSTANTS
PIPI_METADATA_JSON_URI = "https://pypi.org/pypi/{}/{}/json"

# PATHS
# The empty file is used to help filter out common imports from the target
EMPTY_FILE_PATH = os.path.join(os.path.dirname(os.path.realpath(__file__)),
                               'resources',
                               'empty_file.py')


def _get_actual_import_from(file_path: str, import_name: str) -> list:
  """Tries to define the full import name for an import.

  Sometimes imports are defined as from x import y, wherein y can be part of the
  canonical wheel name (this is more common in google.cloud libraries than
  elsewhere). For those cases, read the source file and find the line(s) with
  the specified import, and find the next string, concatenating it to the
  original to get the full name. This requirement is due to an issue with
  modulefinder (wherein it doesn't realize this is the fully qualified name).

  Args:
    file_path: Full path to the source file.
    import_name: The name of the import (before the from).
  Returns:
    The fully qualified name if any.
  """
  logging.debug("Trying to get actual name for {}".format(import_name))
  result = []
  with open(file_path, 'r') as target_script:
    for line in target_script.readlines():
      if 'from {} import'.format(import_name) in line:
        result.append("{}.{}".format(import_name, line.split(" ")[-1].strip()))
  return result


def _parse_requirement(raw_requirement_string: str) -> gopo.Dependency:
  """Use the tokenizer to get the fully qualified Dependency."""
  tokens = tokenizer.DependencyTokenizer(raw_requirement_string)

  library_name = tokens.parse_name()
  versions = tokens.parse_versions()
  metadata = tokens.parse_metadata()

  return gopo.Dependency(library_name, versions, metadata)


def find_libraries(library_name: str,
                   potential_wheels: wheels.Wheel,
                   source_file: str) -> list:
  """Find a list of libraries for a given library_name in potential_wheels.

  Tries to match a name to what is available in potential wheels. Since some
  imports are layered (e.g.: a.b.c), the python tool to determine the actual
  import is unable to retrieve them. As such we try to glean manually what
  those imports are in those cases: in those cases we find the closest name
  that fully matches, or don't add the wheel if not (logging a warning in
  that case).

  Args:
    library_name: Name of the library to find dependencies for.
    potential_wheels: The wheel library to look for libraries in.
    source_file: Path to the source code (necessary if modulfinder fails).
  Returns:
    A list of Dependency objects that match the requirement.
  """
  # For top level libraries only
  potential_wheel = potential_wheels.match(
      gopo.Dependency.clean_library_name(library_name))
  if potential_wheel:
    return [gopo.Dependency(potential_wheel)]
  possible_actual_libraries = _get_actual_import_from(source_file, library_name)
  if possible_actual_libraries:
    result = []
    for current_library in possible_actual_libraries:
      potential_wheel = potential_wheels.match(
          gopo.Dependency.clean_library_name(current_library))
      if potential_wheel:
        result.append(gopo.Dependency(potential_wheel))
    if result:
      return result
  logging.warning("Could not find wheel for {}".format(library_name))
  return []


def find_dependencies(library: str,
                      version: gopo.Version,
                      python_version: gopo.Version,
                      with_extras: list=[]) -> list:
  """ Query pypi for relevant dependencies.

  Query https://pypi.org/pypi/<lib>/<version>/json for [info][requires_dist],
  adding any extras in brackets if any.

  Args:
    library: Library to query in pypi
    version: Version of the library to query in pypi
    python_version: Filter for dependencies (from dep metadata)
    with_extras: Filter for dependencies (for dep metadata).
  Returns:
    A list of matching gopo.Dependency.
  """
  dependencies = []

  metadata_uri = PIPI_METADATA_JSON_URI.format(library, version)
  logging.info("Querying URI: {}".format(metadata_uri))

  # Don't want to have to install requests
  unparsed_library_metadata = urllib.request.urlopen(
      metadata_uri
  ).read()

  # Unfortunately None is hardcoded as a string, so we have to do a little
  # trickery to work around that (normally we would just use .get(..., []):
  parsed_requirements = json.loads(unparsed_library_metadata)\
      .get("info", {})\
      .get("requires_dist", []) or []

  for requirement in parsed_requirements:
    dependency = _parse_requirement(requirement)

    if dependency.metadata.get('extra'):
      if dependency.metadata['extra']['value'] not in with_extras:
        continue

    if dependency.metadata.get('python_version'):
      operation = utilities.OPERATIONS_MAP[dependency.metadata['python_version']['comparator']]
      candidate_version = gopo.Version(dependency.metadata['python_version']['value'])
      if not operation(python_version, candidate_version):
        continue

    dependencies.append(dependency)

  logging.debug("Found dependencies: {}".format(dependencies))
  return dependencies


def parse_arguments() -> argparse.Namespace:
  """Simple ArgParse populator."""
  parser = argparse.ArgumentParser(description='Find all vpython dependencies for a python file.')
  parser.add_argument('--input', '-i',
                      action='store',
                      required=True,
                      help="Python file which needs definitions determined")
  parser.add_argument('--verbose', '-v',
                      action='count',
                      default=0,
                      help="Cumulative logging level")
  parser.add_argument('--use-cache', '-c',
                      action='store_true', default=False)
  parser.add_argument('--python', '-p',
                      default='3.8')
  parser.add_argument('--output', '-o',
                      action='store',
                      help=".vpython output file path")
  return parser.parse_args()


def print_output(wheels_list: list,
                 python_version: gopo.Version,
                 output_file_path: str) -> None:
  """Outputs the .vpython definition to stdout or file, dependant on inputs."""
  output = "python_version: \"{}\"\n".format(python_version.get_relevant_version())
  for current_wheel in sorted(wheels_list, key=lambda x: x['name']):
    output += "\nwheel <\n"
    output += "\tname: \"{}\"\n".format(current_wheel['name'])
    output += "\tversion: \"{}\"\n".format(current_wheel['version'])
    output += ">\n"
  if output_file_path:
    with open(output_file_path, 'w') as output_file:
      output_file.write(output)
  else:
    print(output)


def get_missing_base_libraries(target_file: str) -> list:
  """Gets the uninstalled libraries for a file."""
  # Unfortunately module finder finds everything, even if empty. So we need to
  # compare to a an empty baseline (created solely for this comparison)
  finder_base = modulefinder.ModuleFinder()
  finder_target = modulefinder.ModuleFinder()

  finder_base.run_script(EMPTY_FILE_PATH)
  finder_target.run_script(target_file)

  return list(set(finder_target.any_missing())
              - set(finder_base.any_missing()))


def main() -> None:
  # First we do basic set-up
  args = parse_arguments()
  logging.basicConfig(format='%(levelname)s: %(message)s',
                      level=utilities.LOGGING_LEVELS.get(
                          args.verbose,
                          logging.DEBUG))

  if not gopo.Version.is_canonical_python_version(args.python):
    sys.stderr.write(
        "Version provided does not match a valid python version: {}\n"
        .format(args.python))
    sys.exit(1)

  # Extract from arguments
  python_version = gopo.Version(args.python)
  input_file = args.input

  # Get available wheels to pick from
  potential_wheels = wheels.Wheel(args.use_cache)
  logging.debug("Available wheels: {}".format(
      pprint.pformat(potential_wheels, indent=2)
  ))

  missing_libraries = get_missing_base_libraries(input_file)
  logging.debug("Missing libraries: {} ".format(missing_libraries))

  iterable_dependencies= []
  visited_dependencies = set()
  wheel_output = []

  # Get valid Dependency names for top-level libraries
  for missing_library in missing_libraries:
    found_libraries = find_libraries(
        missing_library, potential_wheels, input_file)
    for found_library in found_libraries:
      if found_library.canonical_name not in visited_dependencies:
        visited_dependencies.add(found_library.canonical_name)
        iterable_dependencies.append(found_library)

  # BFS
  while(iterable_dependencies):
    current_dependency = iterable_dependencies.pop()
    potential_hit = potential_wheels.match(current_dependency.canonical_name)
    found_version, found_wheel = potential_wheels.get(potential_hit)\
        .find_best_fit(python_version.get_major_version(),
                       current_dependency.versions)
    wheel_output.append(found_wheel)
    new_potential_dependencies = find_dependencies(current_dependency.library_name,
                                                   found_version,
                                                   python_version)

    for dependency in new_potential_dependencies:
      # TODO if it is in, we should still check if there's some newer definition
      # and update accordingly (this part might get complicated)
      if dependency.canonical_name not in visited_dependencies:
        visited_dependencies.add(dependency.canonical_name)
        iterable_dependencies.append(dependency)

  # Start printing the output
  logging.info("Found {} wheels".format(len(wheel_output)))
  print_output(wheel_output, python_version, args.output)


if __name__ == '__main__':
  main()
