unit {
  v_name {
    signature: "#2186a1be811eaf56d7637f18695b2e2f9ecd4fa05439d8e925f21e7bc44a8693"
    language: "java"
  }
  required_input {
    v_name {
      corpus: "chromium-test"
      path: "/source/chromium/src/build/android/buildhooks/java/org/chromium/build/BuildHooks.java"
    }
    info {
      path: "/source/chromium/src/build/android/buildhooks/java/org/chromium/build/BuildHooks.java"
      digest: "83f9288e07d620c6bbdce07ec5feb36a5c4488e4a2ea536dd89cfcc3cc726ec8"
    }
  }
  required_input {
    v_name {
      corpus: "chromium-test"
      path: "gen/build/android/buildhooks/build_hooks_java/generated_java/input_srcjars/org/chromium/build/BuildHooksConfig.java"
    }
    info {
      path: "gen/build/android/buildhooks/build_hooks_java/generated_java/input_srcjars/org/chromium/build/BuildHooksConfig.java"
      digest: "6a44da05e4dd2fe42a8678f33354abdcee87a8549ef4d0f6ae68d82a9d95285b"
    }
  }
  argument: "--boot-class-path"
  argument: "/chromium_code"
  source_file: "/source/chromium/src/build/android/buildhooks/java/org/chromium/build/BuildHooks.java"
  source_file: "gen/build/android/buildhooks/build_hooks_java/generated_java/input_srcjars/org/chromium/build/BuildHooksConfig.java"
  output_key: "gen/build/android/buildhooks/build_hooks_java.javac.jar.staging/classes"
  working_directory: "/source/chromium/src/out/xxx1"
  details {
    [kythe.io/proto/kythe.proto.JavaDetails] {
      sourcepath: "/source/chromium/src/build/android/buildhooks/java"
      sourcepath: "gen/build/android/buildhooks/build_hooks_java/generated_java/input_srcjars"
    }
  }
  details {
    [kythe.io/proto/kythe.proto.BuildDetails] {
      build_target: "#2186a1be811eaf56d7637f18695b2e2f9ecd4fa05439d8e925f21e7bc44a8693"
      build_config: "linux"
    }
  }
}
