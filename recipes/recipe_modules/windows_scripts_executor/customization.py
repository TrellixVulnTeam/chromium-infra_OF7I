# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
from . import add_windows_package
from . import add_windows_driver

class Customization(object):
  """ Base customization class. Provides support for pinning and executing
      recipes.
  """

  def __init__(self, arch, scripts, configs, step, path, powershell, m_file,
               source):
    """ __init__ copies common module objects to class references. These are
        commonly used for all customizations
        Args:
          arch: string representing architecture to build the image for
          scripts: path to the scripts resource dir
          step: module object for recipe_engine/step
          path: module object for recipe_engine/path
          powershell: module object for recipe_modules/powershell
          m_file: module object for recipe_engine/file
          source: module object for Source from sources.py
    """
    self._arch = arch
    self._scripts = scripts
    self._step = step
    self._path = path
    self._powershell = powershell
    self._source = source
    self._file = m_file
    self._key = ''
    self._configs = configs
    self._name = ''

  def name(self):
    """ name returns the name of the customization object. This needs to be
        set by the inheriting class"""
    return self._name

  def set_key(self, key):
    """ set_key is used to set the identification keys for the customization
        Args:
          key: string representing the unique key for this customization
    """
    self._key = key

  def execute_script(self, name, command, logs=None, *args):
    """ Executes the windows powershell script
        Args:
          name: string representing step name
          command: string|path representing command to be run
          logs: list of strings representing log files/folder to be read
          args: args to be passed on to the command
    """
    return self._powershell(name, command, logs=logs, args=list(args))
