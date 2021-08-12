create {
  platform_re: ".*-386"
  unsupported: true
}

create {
  source { git {
    repo: "https://chromium.googlesource.com/external/github.com/ninja-build/ninja"
    tag_pattern: "v%s"
  }}
}

create {
  platform_re: "mac-.*|linux-amd64"
  build {
    tool: "tools/re2c"
  }
}

create {
  platform_re: "linux-arm.*|linux-mips.*"
  build {
    tool: "tools/ninja"  # Depend on the bootstrapped version when cross-compiling
    tool: "tools/re2c"
  }
}

create {
  platform_re: "windows-.*|mac-.*|linux-amd64"
  build {
    install: "install_bootstrap.sh"
  }
}

upload { pkg_prefix: "tools" }
