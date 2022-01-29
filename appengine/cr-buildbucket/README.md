# Buildbucket

Buildbucket is a generic build queue. A build requester can schedule a build
and wait for a result. A building system, such as Swarming, can lease it, build
it and report a result back.

*   Documentation: [go/buildbucket](http://go/buildbucket).
    TODO(nodir): add a link to exported public doc when available.
*   Original design doc and V1 API documentation: [go/buildbucket-design](http://go/buildbucket-design)
*   Deployments:
    *   Prod: [cr-buildbucket.appspot.com](https://cr-buildbucket.appspot.com) [[API](https://cr-buildbucket.appspot.com/rpcexplorer/services/buildbucket.v2.Builds/)]
    *   Dev: [cr-buildbucket-dev.appspot.com](https://cr-buildbucket-dev.appspot.com) [[API](https://cr-buildbucket-dev.appspot.com/rpcexplorer/services/buildbucket.v2.Builds/)]
*   Bugs: [Infra>LUCI>BuildService>Buildbucket component](https://crbug.com?q=component:Infra>LUCI>BuildService>Buildbucket)
*   Contact: luci-team@

## Rewrite

Currently Buildbucket is in the middle of a service rewrite in Go. The new code
is found under [luci-go](https://chromium.googlesource.com/infra/luci/luci-go/+/refs/heads/main/buildbucket/appengine/)
and is intended to support an in-place migration from the Python service located
here.

* Umbrella bug: [crbug.com/1042991](https://crbug.com/1042991).
* Project review record:
  [crbug.com/chrome-operations/26](https://crbug.com/chrome-operations/26).

### Deployment

The [service configuration](./app.yaml) for Buildbucket is located in this
directory, but the [cron
job](https://chromium.googlesource.com/infra/luci/luci-go/+/refs/heads/main/buildbucket/appengine/frontend/cron.yaml)
and [task queue](https://chromium.googlesource.com/infra/luci/luci-go/+/refs/heads/main/buildbucket/appengine/frontend/queue.yaml)
configs are found in luci-go. All Python and Go GAE services must be uploaded
for Buildbucket to function. [dispatch.yaml](./dispatch.yaml) defines which HTTP
routes are served by the Go AppEngine service (known as `default-go`).

Deployment of `cr-buildbucket` requires
[gae.py](https://chromium.googlesource.com/infra/luci/luci-py/+/refs/heads/main/appengine/components/tools/gae.py)
similar to other LUCI services whose deployment has not yet been automated. The
Buildbucket Python services (`default`, `backend`, `beefy`) are deployed from
this directory. To deploy the Go service (`default-go`), see the dummy
[app.yaml](https://chromium.googlesource.com/infra/luci/luci-go/+/refs/heads/main/buildbucket/appengine/frontend/app.yaml)
file for instructions. Note that `cr-buildbucket-dev` has its Python services
[automatically deployed](https://chrome-internal.googlesource.com/infradata/gae/+/refs/heads/main/apps/cr-buildbucket/).

* In this dir: `gae.py upload -A cr-buildbucket{,-dev}`
* In
  [luci-go](https://chromium.googlesource.com/infra/luci/luci-go/+/refs/heads/main/buildbucket/appengine/frontend/):
  `gae.py upload -A cr-buildbucket{,-dev} default-go`
