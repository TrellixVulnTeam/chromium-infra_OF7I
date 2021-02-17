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
docs](https://chromium.googlesource.com/infra/ijfra//+/HEAD/go/README.md#get-the-code).

To verify that things are set up correctly, execute `make` from this directory:

      $ make all
      $ make test
