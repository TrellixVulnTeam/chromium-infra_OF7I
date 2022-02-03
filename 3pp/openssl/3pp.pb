create {
  platform_re: "linux-.*|mac-.*"
  source {
    url {
      download_url: "https://www.openssl.org/source/openssl-1.1.1j.tar.gz"
      version: "1.1.1j"
    }
    patch_version: "chromium.2"
    patch_dir: "patches"
    unpack_archive: true
    cpe_base_address: "cpe:/a:openssl_project:openssl"
  }
  build {}
}

upload { pkg_prefix: "static_libs" }
