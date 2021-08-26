# Weetbix

Weetbix is a system designed to understand and reduce the impact of test
failures.

This app follows the structure described in [Template for GAE Standard app].

## Deployment

Weetbix uses `gae.py` for deployment.

First, enter the infra env (via the infra.git checkout):
```
eval infra/go/env.py
```

Then use the following commands to deploy:
```
./gae.py upload -A <appid>
./gae.py switch -A <appid>
```

Currently, the appid is chops-weetbix-dev (for dev) or chops-weetbix (for prod).

[Template for GAE Standard app]: https://chromium.googlesource.com/infra/luci/luci-go/+/HEAD/examples/appengine/helloworld_standard/README.md
