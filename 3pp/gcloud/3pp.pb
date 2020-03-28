create {
  platform_re: "linux-amd64|mac-amd64"
  source {
    script { name: "fetch.py" }
    unpack_archive: true
    no_archive_prune: true
    patch_dir: "patches"
    patch_version: "chromium1"
  }
  build {
    no_toolchain: true
  }
}

upload { pkg_prefix: "tools" }
