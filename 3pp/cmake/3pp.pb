create {
  platform_re: ".*-386"
  unsupported: true
}

create {
  platform_re: "linux-.*|mac-.*"
  source {
    git {
      repo: "https://chromium.googlesource.com/external/github.com/Kitware/CMake"
      tag_pattern: "v%s"

      # TODO: This restriction is in place because the Docker containers we
      # currently use are aimed at 'manylinux1' python compatibility, which
      # has an INCREDIBLY old version of libc. Newer versions of libuv (a
      # dependency of cmake) drop support for this.
      #
      # This restriction prevents trying to build anything from the 3.14 and
      # up family.
      #
      # Upstream bug: https://gitlab.kitware.com/cmake/cmake/issues/19086
      #
      # The fix for US is to switch from manylinux1 to manylinux2010 (or
      # newer).
      version_restriction: { op: LT val: "3.14rc0"}
    }
  }

  build {
    tool: "cmake_bootstrap"
    tool: "ninja"
  }
}

upload { pkg_prefix: "tools" }
