create {
  platform_re: "linux-.*|mac-.*"
  source {
    script { name: "fetch.py" }
    unpack_archive: true
    patch_version: "chromium.1"
  }

  build {}
}

upload { pkg_prefix: "tools" }