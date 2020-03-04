#
# See the pRPC spec here: https://godoc.org/github.com/luci/luci-go/grpc/prpc
#
# Each Servicer corresponds to a service defined in a .proto file in this
# directory. Each method on that Servicer corresponds to one of the rpcs
# defined on the service.
#
# All APIs are served under the /prpc/* path space. Each service gets its own
# namespace under that, and each method is an individual endpoints. For example,
# POST https://bugs.chromium.org/prpc/monorail.v1.Issues/GetIssue
# would be a call to the api.v1.issues_servicer.IssuesServicer.GetIssue method.
#
# Note that this is not a RESTful API, although it is CRUDy. All requests are
# POSTs, all methods take exactly one input, and all methods return exactly
# one output.
#
# TODO(agable): Actually integrate the rpcexplorer.
# You can use the API Explorer here: https://bugs.chromium.org/rpcexplorer

from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

from api.v1 import issues_servicer
from api.v1 import hotlists_servicer
from api.v1 import projects_servicer


def RegisterApiHandlers(prpc_server, services):
  """Registers pRPC API services. And makes their routes
  available in prpc_server.get_routes().
  """

  prpc_server.add_service(issues_servicer.IssuesServicer(services))
  prpc_server.add_service(hotlists_servicer.HotlistsServicer(services))
  prpc_server.add_service(projects_servicer.ProjectsServicer(services))
