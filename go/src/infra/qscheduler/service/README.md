# QScheduler-Swarming

QScheduler-Swarming is an implementation of the [ExternalScheduler](https://chromium.googlesource.com/infra/luci/luci-py/+/refs/heads/master/appengine/swarming/proto/api/plugin.proto) API for [swarming](https://chromium.googlesource.com/infra/luci/luci-py/+/refs/heads/master/appengine/swarming/), using the [quotascheduler](https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/qscheduler/qslib/) algorithm.

This server runs on Kubernetes.

Code layout:

- `api/`            Definitions for the Admin API to QScheduler-Swarming.
- `app/config`      Definitions for QScheduler-Swarming configuration.
- `app/eventlog`    BigQuery logging helper.
- `app/frontend`    Request handlers.
- `app/state`       Common code for mutating scheduler state, and request-batcher implementation.
- `app/state/metrics`       Time-series and analytics emission.
- `app/state/nodestore`     Datastore-backed store of scheduler state with in-memory cache, distributed over many write-only datastore entities.
- `app/state/operations`    State mutators for scheduling requests.
- `cmd/qscheduler-swarming` Entry point for server.
- `docker`                  Context directory for Docker (including Dockerfile)


## Making changes

Submitted changes are automatically deployed to the staging instance by
[Chrome Ops Kubernetes] automation. It usually takes 5-10 min. To quickly check
what version is currently running, visit [/healthz] endpoint and look at the
image tag. It looks like `ci-2020.04.10-30467-b307348`. The last part is a
git revision of infra.git repository used to build the image from.

Once a change is verified on the staging, land a [channels.json] CL that makes
`stable` version equal to the current `staging` version there. It will result
in the production cluster update.

If you want to deploy an uncommitted change to the staging cluster, follow
instructions in [dev/k8s.star]. Note that there's only one staging instance
and you'll be fighting with the automation to deploy changes to it (unless
the automation is disabled).

To make changes to Kubernetes configuration, modify files in projects/qscheduler
in [k8s.git] repo. Changes are applied on commit. See [Chrome Ops Kubernetes]
for all details.

[Chrome Ops Kubernetes]: https://chrome-internal.googlesource.com/infradata/k8s/+/refs/heads/master/README.md
[/healthz]: https://qscheduler-dev.chromium.org/healthz
[channels.json]: https://chrome-internal.googlesource.com/infradata/k8s/+/refs/heads/master/projects/qscheduler/channels.json
[dev/k8s.star]: https://chrome-internal.googlesource.com/infradata/k8s/+/refs/heads/master/projects/qscheduler/dev/k8s.star
[k8s.git]: https://chrome-internal.googlesource.com/infradata/k8s/+/refs/heads/master/projects/qscheduler
