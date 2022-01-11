create {
  platform_re: "linux-.*|mac-.*"
  source {
    url {
      download_url: "https://archive.apache.org/dist/apr/apr-util-1.6.1.tar.gz"
      version: "1.6.1"
    }
    unpack_archive: true
  }

  build {
    dep: "static_libs/apr"
  }
}

upload { pkg_prefix: "static_libs" }
