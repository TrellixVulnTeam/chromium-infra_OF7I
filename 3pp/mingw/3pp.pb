create {
  platform_re: "windows-.*"
  source {
    script { name: "fetch.py" }
    unpack_archive: true
  }
}

upload { pkg_prefix: "build_support" }
