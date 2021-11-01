# Chromeperf Services

This directory contains source code for all new services for the Chromeperf
project. This will include:

-   Microservices for the Pinpoint project.
-   Microservices for the Chromeperf Dashboard.

Code for existing services for the Chromeperf project are currently hosted in
the
[catapult](https://source.chromium.org/chromium/chromium/src/+/master:third_party/catapult/)
repository.

## Onboarding Steps

The only thing you need to do is to just run the generic bootstrapping that is
needed for the Go infra directory, as noted by [these other
docs](https://chromium.googlesource.com/infra/infra//+/HEAD/go/README.md#get-the-code).

To verify that things are set up correctly, execute `make` from this directory:

      $ make all
      $ make test

## Deploying chromeperf/pinpoint/server

After a change is pushed to Geritt, a build will be executed [here](https://ci.chromium.org/p/infra-internal/builders/prod/infra-docker-images-continuous), and metadata for successful builds will automatically be added [here](https://source.corp.google.com/chops_infra_internal/data/k8s/images/gcr.io/chops-public-images-prod/chromeperf/pinpoint_server/). To deploy a build, find the metadata corresponding to your change, and modify the stable and canary lines in the pinpoint_server block [here](https://source.corp.google.com/chops_infra_internal/data/k8s/projects/chromeperf/channels.json).

For the more detailed version of this explanation, see [here](https://source.corp.google.com/chops_infra_internal/data/k8s/README.md).