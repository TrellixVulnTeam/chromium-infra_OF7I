# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.


# Emulate the bare minimum for chardet for monorail.
# In practice, we do not need it, and it's very large.
__version__ = '3.0.2'
def detect(_ignored):
  return {'encoding': 'utf-8'}
