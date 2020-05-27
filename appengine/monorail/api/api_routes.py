# Copyright 2016 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

# This file implements a pRPC API for Monorail.
#
# See the pRPC spec here: https://godoc.org/github.com/luci/luci-go/grpc/prpc
#
# Each Servicer corresponds to a service defined in a .proto file in this
# directory. Each method on that Servicer corresponds to one of the rpcs
# defined on the service.
#
# All APIs are served under the /prpc/* path space. Each service gets its own
# namespace under that, and each method is an individual endpoints. For example,
#   POST https://bugs.chromium.org/prpc/monorail.Users/GetUser
# would be a call to the api.users_servicer.UsersServicer.GetUser method.
#
# Note that this is not a RESTful API, although it is CRUDy. All requests are
# POSTs, all methods take exactly one input, and all methods return exactly
# one output.
#
# TODO(http://crbug.com/monorail/1703): Actually integrate the rpcexplorer.
# You can use the API Explorer here: https://bugs.chromium.org/rpcexplorer

from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

from api import features_servicer
from api import issues_servicer
from api import projects_servicer
from api import sitewide_servicer
from api import users_servicer


def RegisterApiHandlers(prpc_server, services):
  """Registers pRPC API services. And makes their routes
  available in prpc_server.get_routes().
  """

  prpc_server.add_service(features_servicer.FeaturesServicer(services))
  prpc_server.add_service(issues_servicer.IssuesServicer(services))
  prpc_server.add_service(projects_servicer.ProjectsServicer(services))
  prpc_server.add_service(sitewide_servicer.SitewideServicer(services))
  prpc_server.add_service(users_servicer.UsersServicer(services))
