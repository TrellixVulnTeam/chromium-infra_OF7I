# Copyright (c) 2012 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import webapp2

import app
import base_page
import gae_ts_mon

class MainAction(base_page.BasePage):

  def get(self):
    self.redirect('https://ci.chromium.org/p/chromium/g/main/console')


# Call initial bootstrap for the app module.
app.bootstrap()
base_page.bootstrap()

# GAE will cache |application| across requests if we set it here.  See
# http://code.google.com/appengine/docs/python/runtime.html#App_Caching for more
# info.
application = webapp2.WSGIApplication([('/.*', MainAction)])
gae_ts_mon.initialize(application)
