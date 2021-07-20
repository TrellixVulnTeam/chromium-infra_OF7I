create {
  platform_re: "linux-.*|mac-.*",
  source {
    url {
      download_url: "https://ftp.gnu.org/pub/gnu/gettext/gettext-0.19.8.tar.gz"
      version: "0.19.8"
    }
    unpack_archive: true
    patch_dir: "patches"
    patch_version: "chromium.1"
  }
  build {}
}

upload { pkg_prefix: "tools" }
