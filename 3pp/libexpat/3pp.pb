 create {
   platform_re: "linux-.*|mac-.*"
   source { git {
     repo: "https://github.com/libexpat/libexpat"
     # libexpat release tags look like "R_1_2_3"
     tag_pattern: "R_%s"
     version_join: "_"
    }
   }
   build {
    tool: "autoconf"
    tool: "automake"
   }
 }

 upload { pkg_prefix: "static_libs" }
