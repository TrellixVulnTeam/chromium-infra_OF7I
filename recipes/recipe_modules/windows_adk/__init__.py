# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

DEPS = [
    'recipe_engine/cipd',
    'recipe_engine/context',
    'recipe_engine/step',
    'recipe_engine/path',
]

from recipe_engine.recipe_api import Property
from recipe_engine.config import ConfigGroup, Single

PROPERTIES = {
    'win_adk_refs':
        Property(
            help='Refs to pull windows adk',
            param_name='win_adk_refs',
            kind=str,
            default='latest'),
    'win_adk_winpe_refs':
        Property(
            help='Refs to pull win-pe add-on',
            param_name='win_adk_winpe_refs',
            kind=str,
            default='latest'),
}
