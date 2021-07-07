create {
  source {
    git {
      repo: "https://github.com/pypa/virtualenv.git"
      version_restriction {
        op: EQ
        val: "16.7.10"
      }
    }
    patch_version: "chromium.1"
  }
  build {
    dep: "build_support/pip_bootstrap"
  }
}

upload {
  pkg_prefix: "tools"
  universal: true
}
