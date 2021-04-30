create {
  platform_re: "linux-armv6l|linux-mips.*"
  unsupported: true
}

create {
  source {
    script { name: "fetch.py" }
    unpack_archive: true
  }
}

upload { pkg_prefix: "tools" }
