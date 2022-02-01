create {
  platform_re: "linux-armv6l|linux-mips.*"
  unsupported: true
  source {
    cpe_base_address: "cpe:/a:nodejs:nodejs"
  }
}

create {
  # mac, windows, linux 64bit, linux arm 32/64
  platform_re: ".*amd64|.*arm.*"
  source {
    script { name: "fetch.py" }
    unpack_archive: true
  }
}

upload {
  pkg_prefix: "tools"
}

