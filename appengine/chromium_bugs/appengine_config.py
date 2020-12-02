# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is govered by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

"""Configuration."""

import os
import sys

# Enable third-party imports
sys.path.append(os.path.join(os.path.dirname(__file__), 'third_party'))

# Add the gae_ts_mon/protobuf directory into the path for the google package, so
# "import google.protobuf" works.
# For some reason the fix in gae_ts_mon.__init__ does not work for this app.
# The differences here are the updated path below, and reload().
protobuf_dir = os.path.join(os.path.dirname(__file__), 'gae_ts_mon', 'protobuf')
import google
google.__path__.append(os.path.join(protobuf_dir, 'google'))
sys.path.insert(0, protobuf_dir)
reload(google)
