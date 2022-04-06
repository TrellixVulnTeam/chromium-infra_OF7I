create {
  # We are currently building this package only for linux platform.
  platform_re: "linux-amd64|linux-arm64"

  source {
    git {
      repo: "https://github.com/google/nsjail"
      tag_pattern: "%s"

      # We would like to use a fixed version of nsjail so that we can keep
      # its config stable in our codebase. Fixed to 3.1 for now.
      version_restriction: { op: EQ val: "3.1"}
    }
    patch_dir: "patches"
    patch_version: "chromium.1"
  }

  build {
    tool: "tools/flex"
    tool: "tools/protoc"
    dep: "tools/protobuf-cpp"
    dep: "tools/libnl"
  }
}

upload { pkg_prefix: "tools" }