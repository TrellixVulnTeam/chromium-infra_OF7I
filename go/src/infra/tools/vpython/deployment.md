# Deployment

`vpython` is deployed using a variety of mechanisms.

Use following commit to generate change list between two versions:

```bash
git --no-pager log --oneline old_commit...new_commit go/src/infra/tools/vpython/

# This change includes commits:
# 92464f71a9 Update to new virtualenv with mac-arm64 native wheel support.
# 4bf15e85d2 Update virtualenv package version.
# a0d5d3afba Adding .vpython generating tool (Proof of Concept)
```

## LUCI builds

NOTE: Because buildbucket has a painless canary support, it's preferred to deploy in buildbucket canary before swarming or puppet.

### Swarming Task Template

`vpython` is deployed in these task templates:

- [Chromium Swarm Production]
- [Chromium Swarm Development]
- [Chrome Swarm Production]

To deploy a new version of `vpython`:

1. Deploy to [Chromium Swarm Development]. Check [task status](https://chromium-swarm-dev.appspot.com/tasklist) in development environment.
3. (Optional) Deploy canary task template to [Chromium Swarm Production] and **CC [primary trooper](https://oncall.corp.google.com/chrome-ops-client-infra)**. Check [task status](https://chromium-swarm.appspot.com/tasklist) in production environment with [`swarming.pool.template-tag:canary`](https://chromium-swarm.appspot.com/tasklist?f=swarming.pool.template-tag%3Acanary).
1. Deploy to [Chromium Swarm Production] and **CC [primary trooper](https://oncall.corp.google.com/chrome-ops-client-infra)**. Check [task status](https://chromium-swarm.appspot.com/tasklist) in development environment.
4. Deploy to [Chrome Swarm Production] and **CC [primary trooper](https://oncall.corp.google.com/chrome-ops-client-infra)**. Check [task status](https://chrome-swarming.appspot.com/tasklist) in production environment.

Swarming task template supports canary but the canary task template is disruptive for some users due to the deduping behavior. It should be only used for risky release. An example configuration:
```
task_template_deployment {
  name: "chrome_packages"

  prod {
    include: "chrome_packages_prod"
    cipd_package {
      path: ".task_template_packages"
      pkg: "infra/tools/luci/vpython-native/${platform}"
      version: "git_revision:0d045343d70a8309ec92c2cc46c21ee90c68344f"
    }
    cipd_package {
      path: ".task_template_packages"
      pkg: "infra/tools/luci/vpython/${platform}"
      version: "git_revision:0d045343d70a8309ec92c2cc46c21ee90c68344f"
    }
  }
  canary {
    # TODO(crbug/1235841): Move this back to the prod template once it is
    # rolled out.
    include: "chrome_packages_prod"
    cipd_package {
      path: ".task_template_packages"
      pkg: "infra/tools/luci/vpython-native/${platform}"
      version: "git_revision:0915c6a38fe8862a3790dd5bcf2b99c92399199f"
    }
    cipd_package {
      path: ".task_template_packages"
      pkg: "infra/tools/luci/vpython/${platform}"
      version: "git_revision:0915c6a38fe8862a3790dd5bcf2b99c92399199f"
    }
  }

  canary_chance: 5000 # 50% chance of picking canary
}
```


[Chromium Swarm Production]: https://chrome-internal.googlesource.com/infradata/config/+/refs/heads/main/configs/chromium-swarm/pools.cfg
[Chromium Swarm Development]: https://chrome-internal.googlesource.com/infradata/config/+/refs/heads/main/configs/chromium-swarm-dev/pools.cfg
[Chrome Swarm Production]: https://chrome-internal.googlesource.com/infradata/config/+/refs/heads/main/configs/chrome-swarming/pools.cfg

### Buildbucket

vpython is deployed in these buildbucket environments:

- [Buildbucket Production]
- [Buildbucket Development]

[Buildbucket Development] is always using the latest `vpython`. We don't need to update it when deploying a new version.
Deploy to [Buildbucket Production] is similar to [Chromium Swarm Production]. The only difference is Buildbucket has a different canary mechanism without task deduplication. So it's ok to use it for every release:
```
swarming {
  milo_hostname: "ci.chromium.org"
  ...
  user_packages {
    package_name: "infra/tools/luci/vpython/${platform}"
    version: "git_revision:0d045343d70a8309ec92c2cc46c21ee90c68344f"
    version_canary: "git_revision:0915c6a38fe8862a3790dd5bcf2b99c92399199f"
  }
  user_packages {
    package_name: "infra/tools/luci/vpython-native/${platform}"
    version: "git_revision:0d045343d70a8309ec92c2cc46c21ee90c68344f"
    version_canary: "git_revision:0915c6a38fe8862a3790dd5bcf2b99c92399199f"
  }
  ...
}
```

[Buildbucket Production]: https://chrome-internal.googlesource.com/infradata/config/+/refs/heads/main/configs/cr-buildbucket/settings.cfg
[Buildbucket Development]: https://chrome-internal.googlesource.com/infradata/config/+/refs/heads/main/configs/cr-buildbucket-dev/settings.cfg

### Puppet

vpython is deployed in the [cipd.yaml](https://source.corp.google.com/chops_infra_internal/puppet/puppetm/etc/puppet/hieradata/cipd.yaml). Use canary to deploy the new version.

```
  infra/tools/luci/vpython:
    package: "infra/tools/luci/vpython/${platform}"
    supported:
      - infra/tools/luci/vpython/linux-386
      - infra/tools/luci/vpython/linux-amd64
      - infra/tools/luci/vpython/linux-arm64
      - infra/tools/luci/vpython/linux-armv6l
      - infra/tools/luci/vpython/linux-mips64
      - infra/tools/luci/vpython/linux-mips64le
      - infra/tools/luci/vpython/linux-mipsle
      - infra/tools/luci/vpython/linux-ppc64
      - infra/tools/luci/vpython/linux-ppc64le
      - infra/tools/luci/vpython/linux-s390x
      - infra/tools/luci/vpython/mac-amd64
      - infra/tools/luci/vpython/mac-arm64
      - infra/tools/luci/vpython/windows-386
      - infra/tools/luci/vpython/windows-amd64
    versions:
      canary: git_revision:5fba4fd94ac8ac6ada59d047474c7a9a37f7f812
      stable: git_revision:b07638c0390a878b41b6ddb5b671da9fd7b6e5c3

```
## Users

### depot_tools

Update [cipd_manifest.txt](https://chromium.googlesource.com/chromium/tools/depot_tools/+/main/cipd_manifest.txt) and run `cipd ensure-file-resolve -ensure-file cipd_manifest.txt`. We don't have a way to gradually deploy the new version to users but at least users can rollback the version themselves (simply checkout an old version of `depot_tools.git`).