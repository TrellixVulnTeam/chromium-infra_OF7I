create {
  platform_re: "linux-armv6l|linux-mips.*"
  unsupported: true
}

create {
  # mac, windows, linux 64bit, linux arm 32/64
  platform_re: ".*amd64|.*arm.*"
  source {
    script { name: "fetch.py" }
    unpack_archive: true
  }
  package {
    disable_latest_ref: true
  }
}

upload {
  pkg_prefix: "tools"
}

