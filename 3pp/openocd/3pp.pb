create {
  platform_re: "(linux|mac)-(amd64|arm64)|windows-amd64"
  source {
    script { name: "fetch.py" }
    unpack_archive: true
    cpe_base_address: "cpe:/a:openocd:open_on-chip_debugger"
  }

  package {
    version_file: ".versions/openocd.version"
  }
}

upload { pkg_prefix: "tools" }
