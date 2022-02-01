create {
  platform_re: "linux-.*|mac-.*"
  source {
    url {
      download_url: "https://github.com/skvadrik/re2c/releases/download/1.1.1/re2c-1.1.1.tar.gz"
      version: "1.1.1"
    }
    unpack_archive: true
    cpe_base_address: "cpe:/a:re2c:re2c"
  }
  build {}
}

upload { pkg_prefix: "tools" }
