# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""This recipe verifies importing of chromium bootstrap protos.

The protos are exported via a symlink in
//recipe/recipe_proto/infra/chromium.
"""

from PB.infra.chromium.chromium_bootstrap import (
    ChromiumBootstrapModuleProperties)


def RunSteps(api):
  del api


def GenTests(api):
  del api
  return []
