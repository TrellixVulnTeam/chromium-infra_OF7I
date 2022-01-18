create {
  platform_re: "linux-.*|mac-.*"
  source {
    url {
      download_url: "https://secure.php.net/distributions/php-7.3.31.tar.gz"
      version: "7.3.31"
    }
    unpack_archive: true
  }

  build {
    dep: "tools/httpd"
    dep: "static_libs/zlib"
  }
}

upload { pkg_prefix: "static_libs" }
