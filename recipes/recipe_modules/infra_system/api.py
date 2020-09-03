# Copyright 2017 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import contextlib

from recipe_engine import recipe_api


class InfraSystemApi(recipe_api.RecipeApi):
  """API for interacting with a provisioned infrastructure system."""

  def __init__(self, properties, **kwargs):
    super(InfraSystemApi, self).__init__(**kwargs)
    self._properties = properties

  @property
  def sys_bin_path(self):
    return self._properties.get('sys_bin_path', (
        'C:\\infra-system\\bin' if self.m.platform.is_win
        else '/opt/infra-system/bin'))
