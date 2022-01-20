create {
  platform_re: "linux-.*|mac-.*"
  source {
    url {
      download_url: "https://archive.apache.org/dist/apr/apr-1.6.5.tar.gz"
      version: "1.6.5"
    }
    unpack_archive: true
    patch_dir: "patches"
    patch_version: "chromium.1"
  }

  build {}
}

upload { pkg_prefix: "static_libs" }
