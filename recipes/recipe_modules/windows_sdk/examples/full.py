# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from recipe_engine.recipe_api import Property

DEPS = [
  'windows_sdk',
  'recipe_engine/platform',
  'recipe_engine/properties',
  'recipe_engine/step',
]


PROPERTIES = {
    'bits': Property(kind=int, default=None),
}


def RunSteps(api, bits):
  with api.windows_sdk(enabled=api.platform.is_win, bits=bits):
    api.step('gn', ['gn', 'gen', 'out/Release'])
    api.step('ninja', ['ninja', '-C', 'out/Release'])


def GenTests(api):
  for platform in ('linux', 'mac', 'win'):
    yield (
        api.test(platform) +
        api.platform.name(platform))

  # Target architecture override.
  yield (api.test('win-x64-32bit') + api.platform('win', 64) +
         api.properties(bits=32))
