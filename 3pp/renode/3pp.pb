create {
  platform_re: "linux-.*"
  source {
    url {
      download_url: "https://dl.antmicro.com/projects/renode/builds/renode-1.11.0+20210306gite7897c1.linux-portable.tar.gz"
      version: "renode-1.11.0+20210306gite7897c1"
    }
    unpack_archive: true
  }
  build {}
}

upload { pkg_prefix: "tools" }
