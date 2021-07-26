create {
  platform_re: "linux-.*"
  source {
    script { name: "fetch.py" }
    unpack_archive: true
    patch_version: "chromium.1"
  }

  build {}
}

upload { pkg_prefix: "tools" }