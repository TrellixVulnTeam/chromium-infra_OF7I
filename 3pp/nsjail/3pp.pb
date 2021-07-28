create {
  platform_re: "linux-arm.*"
  unsupported: true
}

create {
  # We are currently building this package only for linux platform.
  platform_re: "linux-.*"

  source {
    git {
      repo: "https://github.com/google/nsjail"
      tag_pattern: "%s"

      # We would like to use a fixed version of nsjail so that we can keep a
      # copy of its config proto in our codebase. Fixed to 3.0 for now.
      version_restriction: { op: EQ val: "3.0"}
    }
    patch_dir: "patches"
    patch_version: "chromium.1"
  }

  build {
    tool: "tools/flex"
    dep: "tools/protobuf-cpp"
    dep: "tools/libnl"
  }
}

upload { pkg_prefix: "tools" }