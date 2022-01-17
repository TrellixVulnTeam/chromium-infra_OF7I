create {
  platform_re: "linux-.*|mac-.*"
  source {
    url {
      download_url: "https://archive.apache.org/dist/httpd/httpd-2.4.38.tar.gz"
      version: "2.4.38"
    }
    unpack_archive: true
    patch_dir: "patches"
    patch_version: "chromium.1"
  }

  build {
    dep: "static_libs/apr"
    dep: "static_libs/apr-util"
    dep: "static_libs/pcre"
    dep: "static_libs/openssl"
  }
}

upload { pkg_prefix: "tools" }
