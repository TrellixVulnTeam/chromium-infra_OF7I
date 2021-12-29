# Drone Queen AppEngine Service

See the [design doc](https://goto.google.com/skylab-drone-containerization).

## Drone queen and drone agent contract

See the [design doc](https://goto.google.com/skylab-drone-containerization).

## Setting up new drone queen instance

Steps for setting up a new instance of drone queen:

1.  Create new GCP project.
2.  Deploy drone queen to App Engine.
3.  Configure the LUCI config service at https://<app-url>/admin/portal.
4.  Add a LUCI config for your app to the LUCI config instance you configured
    above.
5.  Enable the Pub/Sub API for your GCP project.
6.  Configure the chrome-infra-auth service at https://<app-url>/admin/portal.
7.  Add your app's App Engine service account to auth service authorized groups.
