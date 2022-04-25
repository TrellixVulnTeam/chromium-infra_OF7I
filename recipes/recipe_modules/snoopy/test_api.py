# Copyright 2022 The Chromium Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from recipe_engine import recipe_test_api

class SnoopyTestApi(recipe_test_api.RecipeTestApi):
  @recipe_test_api.mod_test_data
  @staticmethod
  def pid(pid):
    """Set the process id for the current test.
    """
    assert isinstance(pid, int), ('bad pid (not integer): %r' % (pid,))
    return pid

  def __call__(self, pid):
    return (self.pid(pid))
