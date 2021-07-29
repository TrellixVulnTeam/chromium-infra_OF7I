# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from recipe_engine import recipe_api
from recipe_engine.recipe_api import Property


class PowershellAPI(recipe_api.RecipeApi):
  """ API to execute powershell scripts """

  def __call__(self, name, command, logs=None, args=None):
    """
    Execute a command through powershell.
    Args:
      * name (str) - name of the step being run
      * command (str|path) - powershell command or windows script/exe to run
      * logs ([]str) - List of logs to read on completion. Specifying dir reads
          all logs in dir
      * args ([]str) - List of args supplied to the command
    Returns:
      Dict containing 'results' as a key
    Raises:
      StepFailure if the failure is detected. See resources/psinvoke.py
    """
    psinvoke = self.resource('psinvoke.py')
    cmd_args = ['--command', command]
    if logs:
      cmd_args += ['--logs'] + logs
    if args:
      cmd_args += ['--'] + args
    results = self.m.python(
        'PowerShell> {}'.format(name),
        psinvoke,
        args=cmd_args,
        stdout=self.m.json.output(),
        ok_ret='any')
    step_results = results.stdout
    if step_results:
      for k, v in step_results.items():
        results.presentation.logs[k] = v
    if not step_results['results']['Success']:
      results = step_results['results']
      raise self.m.step.StepFailure('Failed {}'.format(
          results['ErrorInfo']['Message']))
    return step_results
