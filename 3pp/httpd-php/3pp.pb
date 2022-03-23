create {
  platform_re: "linux-amd64|mac-.*"
  source {
    script { name: "fetch.py" }
    unpack_archive: false
  }
  build {
    tool: "tools/autoconf"
  }
}

upload { pkg_prefix: "tools" }
