create {
  platform_re: "linux-.*|mac-.*"
  source {
    script { name: "fetch.py" }
    unpack_archive: true
  }

  build {}
}

upload { pkg_prefix: "tools" }