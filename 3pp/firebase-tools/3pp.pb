create {
  # We only have native (no_docker_env) support on amd64 hosts on our build
  # system.
  platform_re: ".*-amd64"

  source {
    script { name: "fetch.py" }
  }
  build {
    tool: "nodejs"

    # Node.js is too new to run under the linux-amd64 docker environment
    # (because that image is based on CentOS5 to conform to PEP 513)
    no_docker_env: true
  }
}

upload {
  pkg_prefix: "npm"
}
