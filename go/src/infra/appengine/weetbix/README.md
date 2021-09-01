# Weetbix

Weetbix is a system designed to understand and reduce the impact of test
failures.

This app follows the structure described in [Template for GAE Standard app].

## Local Development

To run the server locally, use:
```
cd frontend
go run main.go -cloud-project chops-weetbix-dev -config-local-dir ../configs
```

Note that `-config-local-dir` is required only if you plan on modifying config
and loading it into Cloud Datastore via the read-config cron job accessible via
http://127.0.0.1:8900/admin/portal/cron for testing. Omitting this, the server
will fetch the current config from Cloud Datastore (as periodically refreshed
from LUCI Config Service).

You may also be able to use an arbitrary cloud project (e.g. 'dev') if you
setup Cloud Datastore emulator and setup a config for that project under
configs.

## Deployment

Weetbix uses `gae.py` for deployment.

First, enter the infra env (via the infra.git checkout):
```
eval infra/go/env.py
```

Then use the following commands to deploy:
```
gae.py upload -A <appid>
gae.py switch -A <appid>
```

Currently, the appid is chops-weetbix-dev (for dev) or chops-weetbix (for prod).

## Run Spanner integration tests using Cloud Spanner Emulator

### Install Cloud Spanner Emulator

#### Linux

The Cloud Spanner Emulator is part of the bundled gcloud, to make sure it's installed:

```
cd infra
gclient runhooks
eval `./go/env.py`
which gcloud # should show bundled gcloud
gcloud components list # should see cloud-spanner-emulator is installed
```

### Run tests

From command line, first set environment variables:

```
export INTEGRATION_TESTS=1
```

Then run go test as usual. For example

```
go test -v ./...
```

[Template for GAE Standard app]: https://chromium.googlesource.com/infra/luci/luci-go/+/HEAD/examples/appengine/helloworld_standard/README.md

