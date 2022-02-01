create {
  platform_re: "linux-.*"
  source {
    script { name: "fetch.py" }
    unpack_archive: true
    patch_version: "chromium.1"
    cpe_base_address: "cpe:/a:protobuf_project:protobuf"
  }

  build {}
}

upload { pkg_prefix: "tools" }
