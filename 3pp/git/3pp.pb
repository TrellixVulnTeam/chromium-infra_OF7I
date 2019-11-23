create {
  verify { test: "git_test.py" }
  source { patch_version: "chromium.1" }
}

create {
  platform_re: "linux-.*|mac-.*"

  source { git {
    repo: "https://chromium.googlesource.com/external/github.com/git/git"
    tag_pattern: "v%s"
  }}

  build {
    tool: "autoconf"
    tool: "sed"
    tool: "gettext"

    dep: "zlib"
    dep: "curl"
    dep: "pcre2"
    dep: "libexpat"
  }
}

create {
  platform_re: "windows-.*"
  source { script { name: "fetch_win.py" }}
  build { install: "install_win.sh" }
}

upload { pkg_prefix: "tools" }
