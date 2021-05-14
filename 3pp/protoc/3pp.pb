create {
  # We will need to switch to building from source to support these platforms.
  platform_re: "linux-armv6l|linux-mips.*|mac-arm64"
  unsupported: true
}

create {
  source {
    script { name: "fetch.py" }
    unpack_archive: true
  }
}

upload { pkg_prefix: "tools" }
