create {
  platform_re: "linux-.*|mac-.*"
  source {
    url {
      # libuuid is part of util-linux
      download_url: "https://mirrors.edge.kernel.org/pub/linux/utils/util-linux/v2.33/util-linux-2.33-rc1.tar.gz"
      version: "2.33-rc1"
    }
    unpack_archive: true
  }
  build {}
}

upload { pkg_prefix: "static_libs" }

