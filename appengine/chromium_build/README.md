# chromium-build

This app is deprecated and mostly dead. It used to serve hacked-up console views
for Buildbot.

These days, it only exists to run a mailer service used by Gatekeeper. When this
is no longer required, this service should be turned down and deleted.

## Deploying

To deploy, just run:

```shell
./gae.py upload -A chromium-build
```

The uploaded version won't start serving production automatically; you'll need
to switch it over in the App Engine console.
