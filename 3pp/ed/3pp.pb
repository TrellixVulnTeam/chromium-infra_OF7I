create {
  platform_re: "linux-.*|mac-.*"
  source {
    cipd {
      pkg: "infra/third_party/source/gnu_ed"
      default_version: "1.15"
      original_download_url: "https://ftp.gnu.org/gnu/ed/"
    }
    unpack_archive: true
  }
  build {}
}

upload { pkg_prefix: "tools" }

