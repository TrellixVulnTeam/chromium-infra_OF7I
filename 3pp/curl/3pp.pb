create {
  platform_re: "linux-.*|mac-.*"

  source {
    url {
      download_url: "https://curl.se/download/curl-7.59.0.tar.gz"
      version: "7.59.0"
    }
    patch_version: "chromium.3"
    unpack_archive: true
    cpe_base_address: "cpe:/a:curl_project:curl"
  }

  build {
    dep: "static_libs/libidn2"
    dep: "static_libs/zlib"
  }
}

create {
  platform_re: "linux-.*"

  build {
    dep: "static_libs/libidn2"
    dep: "static_libs/openssl"
    dep: "static_libs/zlib"
  }
}

upload { pkg_prefix: "static_libs" }
