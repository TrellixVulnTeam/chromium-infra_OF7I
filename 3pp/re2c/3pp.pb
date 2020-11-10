create {
  platform_re: "linux-.*|mac-.*"
  source {
    url {
      download_url: "https://github.com/skvadrik/re2c/archive/1.1.1.tar.gz"
      version: "1.1.1"
    }
    unpack_archive: true
  }
  build {}
}

upload { pkg_prefix: "tools" }
