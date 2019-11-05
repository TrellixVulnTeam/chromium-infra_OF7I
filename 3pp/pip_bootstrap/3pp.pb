create {
  source {
    script { name: "fetch.py" }
    patch_version: "chromium1"
  }
  build {
    no_docker_env: true
  }
}

upload {
  pkg_prefix: "build_support"

  # TODO(crbug.com/914572): enable this again.
  # pip_bootstrap.py created on windows does not have +x bit.
  # universal: true
}
