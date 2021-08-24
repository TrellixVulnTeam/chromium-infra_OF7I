# pylint: skip-file
import json
import logging
import os
import pprint
import subprocess
import tempfile

from libraries import Library

"""
The Wheel class holds an equivalent representation of a Wheel definition.
This involves retrieving and parsing the global wheel definition from git.
This class also provides simple caching of that global definition if
requested. The only stored item is the parsed wheel definition.
"""
class Wheel:
  _FIRST_MATCH = 0
  _WHEELS_KEY = 'wheels'
  _NAME_KEY = 'name'
  _VERSION_KEY = 'version'
  _VERSION_SEPARATOR = ':'
  _VERSION_POSITION = 1
  _URI_SEPARATOR = '/'
  _NAME_POSITION = 3
  _UNIVERSAL_KEY = 'universal'
  _UNIVERSAL_IDENTIFIER = '_py2_py3'
  _PYTHON_2_KEY = '2'
  _PYTHON_2_IDENTIFIER = '_py2'
  _PYTHON_3_KEY = '3'
  _PYTHON_3_IDENTIFIER = '_py3'
  _EXTERNAL_SEPARATOR = '-'
  _CANONICAL_SEPARATOR = '_'
  _CACHE_PATH = "{}/.vpython_wheels".format(os.path.expanduser('~'))
  _GIT_REPO = "https://chromium.googlesource.com/infra/infra"
  _GIT_MD_FILE_PATH = "{}/infra/tools/dockerbuild/wheels.md"

  def __init__(self, use_cache: bool=False):
    self.wheels = self._build_parsed_wheel_dict(use_cache)

  def match(self, library_name: str) -> str:
    """Retrieve the wheel name which best matches the desired library_name.

    This method makes use of two matchers: an exact matcher (i.e.: name is an
    exact match to the wheel name), and a more complex matcher if the first
    fails. If no match exists, return None.

    Args:
      library_name: The canonical name of the library to search for.
    Returns:
      The relevant wheel name to the library_name or None if no match is found.
    """
    logging.info("Finding a match for: {}".format(library_name))
    if self.wheels.get(library_name):
      return library_name
    # TODO pick one with Levenshtein distance
    matches = self._match(library_name)
    return matches[Wheel._FIRST_MATCH] if matches else None

  def get(self, library_name: str) -> str:
    """Get a library wheel name if it exactly matches, or None if not."""
    return self.wheels.get(library_name)

  def _match(self, library_name: str) -> list:
    """Get list of wheel names which best matches the input name.

    Iterate through all wheels for all wheels which have the same initial
    characters. Ultimately we would prefer to de levenshtein distance here,
    but doing this for now.

    Args:
      library_name: The canonical name of the library to match.
    Returns:
      A list of names which best match library_name.
    """
    results = []
    for potential_wheel in self.wheels:
      if potential_wheel.startswith(library_name):
        results.append(potential_wheel)
    return results

  @staticmethod
  def get_unprocessed_wheel_dict(use_cache: bool) -> dict:
    """Returns a dictionary of wheels (pre-processed).

    Retrieves wheel metadata from git and parses that into a pre-processed
    dictionary, saving the results into the cached file. If use_cache is set to
    true (and cache exists), retrieve the value from that cached file instead.

    Args:
      use_cache: Whether to retrieve the dict from local file instead of git.
    Returns:
      A pre-processed dictionary representation of the available wheels.
    """
    if use_cache and os.path.exists(Wheel._CACHE_PATH):
      logging.info("Using local wheel cache.")
      with open(Wheel._CACHE_PATH, 'r') as wheel_file:
        return json.load(wheel_file)
    wheels_dict = Wheel._get_unprocessed_wheel_dict()
    with open(Wheel._CACHE_PATH, 'w') as wheel_file:
      wheel_file.write(json.dumps(wheels_dict))
    return wheels_dict

  @staticmethod
  def _get_unprocessed_wheel_dict() -> dict:
    """Returns a pre-processed dictionary of wheels from git.

    Pull wheel metadata from git into a temporary directory, doing some simple
    parsing to generate an iterable dict.

    Returns:
      A parsed (though pre-processed dictionary) representation of the wheels.
    """
    with tempfile.TemporaryDirectory() as temporary_directory:
      Wheel._pull(temporary_directory)
      md_file_path = Wheel._GIT_MD_FILE_PATH.format(temporary_directory)
      return Wheel._parse_wheels_md(md_file_path)

  @staticmethod
  def _build_parsed_wheel_dict(use_cache: bool) -> dict:
    """Gets and processes the wheel dictionary for programmatic use.

    Gets the pre-processed wheel dictionary from either git or local cache,
    and processes it into a more usable dictionary.

    Args:
      use_cache: Whether to fetch the pre-processed wheel dictionary locally.
    Returns:
      A processed wheel dictionary, keyed by library name.
    """
    wheel_dict = Wheel.get_unprocessed_wheel_dict(use_cache)
    result = {}

    for item in wheel_dict[Wheel._WHEELS_KEY]:
      base_name, py_type = Wheel._extract_real_name_and_type(item[Wheel._NAME_KEY])
      version = Wheel._extract_version(item[Wheel._VERSION_KEY])

      result.setdefault(base_name, Library(base_name))
      result[base_name].add_version(py_type, version, item)

    return result

  @staticmethod
  def _parse_wheels_md(wheels_file: str) -> dict:
    """Reads a wheels metadata file, building a basic dict representation.

    Opens the designated file and reads line by line, parsing with:
      * "wheel: <":            Wheel start tag.
      * ">":                   Wheel end tag.
      * "  name: $name":       The library name.
      * "  version: $version": The version number.

    Args:
      wheels_file: Absolute path to the file containing the wheel metadata.
    Returns:
      A pre-processed dict of wheel information.
    """
    with open(wheels_file, 'r') as wheels_file:
      results = []
      for line in wheels_file:
        if line.startswith('wheel: <'):
          current_result = {}
        if line.startswith('  name:') or line.startswith('  version:'):
          parsed_line = line.split()
          current_result[parsed_line[0][:-1]] = parsed_line[1][1:-1]
        if line.startswith('>'):
          results.append(current_result)

      return {Wheel._WHEELS_KEY: results}

  @staticmethod
  def _pull(output_directory: str) -> None:
    """Literally git pulls the wheel repo to the designated directory."""
    logging.info("Pulling wheels from git to {}.".format(output_directory))
    cmd = 'git clone {} {}'.format(Wheel._GIT_REPO, output_directory)
    subprocess.call(cmd, shell=True,
                    stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    logging.info("Git pull done!")

  @staticmethod
  def _extract_version(version: str) -> str:
    """Parses version from a wheel (different then a pip version)."""
    return version.split(Wheel._VERSION_SEPARATOR)[Wheel._VERSION_POSITION]

  @staticmethod
  def _extract_real_name_and_type(name: str) -> (str, str):
    """Parses the canonical name and python version from a wheel name.

    This method takes a wheel name and sanitizes it by using only an underscore
    as separator; it also extracts frim the end of the name the python version,
    as it matches each specified identifier.

    Args:
      name: The raw wheel library name.
    Returns:
      A tuple of canonical name and python version.
    """
    base_name = name.split(Wheel._URI_SEPARATOR)[Wheel._NAME_POSITION].replace(
        Wheel._EXTERNAL_SEPARATOR,
        Wheel._CANONICAL_SEPARATOR)
    py_type = Wheel._UNIVERSAL_KEY

    if base_name.endswith(Wheel._UNIVERSAL_IDENTIFIER):
      base_name = base_name[:-len(Wheel._UNIVERSAL_IDENTIFIER)]
    elif base_name.endswith(Wheel._PYTHON_2_IDENTIFIER):
      base_name = base_name[:-len(Wheel._PYTHON_2_IDENTIFIER)]
      py_type = Wheel._PYTHON_2_KEY
    elif base_name.endswith(Wheel._PYTHON_3_IDENTIFIER):
      base_name = base_name[:-len(Wheel._PYTHON_3_IDENTIFIER)]
      py_type = Wheel._PYTHON_3_KEY

    return base_name, py_type

  def __str__(self) -> str:
    return str(pprint.pformat(self.wheels))

  def __repr__(self) -> str:
    return self.__str__()


