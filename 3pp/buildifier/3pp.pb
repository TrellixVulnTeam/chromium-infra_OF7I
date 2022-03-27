create {
  source {
    git {
      repo: "https://chromium.googlesource.com/external/github.com/bazelbuild/buildtools"
    }
  }

  build { tool: "tools/go" }

  package {
    version_file: ".versions/buildifer.version"
  }
}

upload { pkg_prefix: "tools" }
