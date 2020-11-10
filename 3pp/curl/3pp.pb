create {
  platform_re: "linux-.*|mac-.*"

  source {
    url {
      download_url: "https://curl.se/download/curl-7.59.0.tar.gz"
      version: "7.59.0"
    }
    patch_version: "chromium.3"
    unpack_archive: true
  }

  build {
    dep: "zlib"
    dep: "libidn2"
  }
}

create {
  platform_re: "linux-.*"

  build {
    dep: "zlib"
    dep: "libidn2"
    dep: "openssl"
  }
}

upload { pkg_prefix: "static_libs" }
