import os

PACKAGE_ROOT_DIR = os.path.dirname(os.path.dirname(__file__))
PACKAGE_LIB_DIR = os.path.dirname(__file__)
PACKAGE_SCRIPTS_DIR = os.path.join(PACKAGE_ROOT_DIR, 'scripts')

PRINT_DEPS_SCRIPT_PATH = os.path.join(PACKAGE_SCRIPTS_DIR, 'print_deps.py')

CACHE_PACKAGES_PATH_TEMPALATE = os.path.join('/', 'tmp',
                                             'package_index_{}.cache')

# Set of packages that should be fine to work with but are not handled properly
# yet.
TEMPORARY_UNSUPPORTED_PACKAGES = {
    # Reason: sys-devel/arc-build fails, but I cannot figure out which package
    # triggers it.
    # Perfectly fine package otherwise.
    'chromeos-base/arc-adbd',
    'chromeos-base/arc-appfuse',
    'chromeos-base/arc-apk-cache',
    'chromeos-base/arc-data-snapshotd',
    'chromeos-base/arc-host-clock-service',
    # A bit strange package with both local sources and aosp url, but should be
    # buildable.
    'chromeos-base/arc-keymaster',
    'chromeos-base/arc-obb-mounter',
    'chromeos-base/arc-sensor-service',
    'chromeos-base/arc-setup',
    'chromeos-base/arcvm-boot-notification-server',
    # A bit strange package using files from platform2/vm_tools, but should be
    # buildable.
    'chromeos-base/arcvm-forward-pstore',
    'chromeos-base/arcvm-mojo-proxy',
    # A bit strange package using files from platform2/camera, but should be
    # buildable.
    'media-libs/arc-camera-profile',
    # Package has BUILD.gn and it does something, but there are no cpp sources.
    # If it can be built but has empty compile_commands, there should be no
    # harm, need to be NO_LOCAL_SOURCE otherwise.
    'chromeos-base/arc-sdcard',

    # TODO: notify owners.
    # Reason: include path is misspelled vs actual dir: nNCache vs
    # https://source.chromium.org/chromiumos/chromiumos/codesearch/+/main:src/aosp/frameworks/ml/driver/cache/nnCache/
    'chromeos-base/aosp-frameworks-ml-nn',
    'chromeos-base/aosp-frameworks-ml-nn-vts',

    # Reason: build dir does not contain out/Debug
    # Is built with Makefile but lists .gn in CROS_WORKON_SUBTREE.
    'chromeos-base/avtest_label_detect',
    # Target //croslog/log_rotator:_log_rotator-install_config has metadata
    # field which makes merge complicated.
    'chromeos-base/bootid-logger',

    # Has cryptohome-flatbuffers-binding gn target which has sources field which
    # is almost the same as the target in chromeos-base/cryptohome except that
    # it uses paths generated for this package.
    'chromeos-base/cryptohome-dev-utils',

    # Reason: sys-cluster/fcp dependency fails build.
    # Perfectly fine package otherwise.
    'chromeos-base/federated-service',

    # Target //u2fd:webauthntool-install_config has metadata field
    # which makes merge complicated.
    'chromeos-base/g2f_tools',

    # Has lorgnette-proxies gn target which has args field which is almost the
    # same as the target in chromeos-base/lorgnette except for one path.
    'chromeos-base/lorgnette_cli',

    # Reason: Include path ./third_party/libuweave/ does not exist.
    # https://source.chromium.org/chromiumos/chromiumos/codesearch/+/main:src/weave/libweave/BUILD.gn;l=29
    'chromeos-base/libweave',

    # Has libmanatee-client-headers gn target which has args field which is
    # almost the same as the target in chromeos-base/vm_host_tools except for
    # one path and one additional arg.
    'chromeos-base/manatee-client',

    # Reason: REQUIRED_USE="minios" fails build.
    # Perfectly fine package otherwise.
    'chromeos-base/minios',

    # Target //ml:_ml_cmdline-install_config has metadata field which makes
    # merge complicated.
    'chromeos-base/ml-cmdline',

    # Reason: /etc/init/ocr_service.conf: missing 'oom score' line
    # Perfectly fine package otherwise.
    'chromeos-base/ocr',

    # Reason: REQUIRED_USE="kvm_guest" fails build.
    # Perfectly fine package otherwise.
    'chromeos-base/sommelier',

    # Reason: override-max-pressure-seccomp-amd64.policy does not exist. Only
    # arm. Not sure if it supposed to be compilable under amd64-generic or need
    # another seccomp.
    'chromeos-base/touch_firmware_calibration',

    # Reason: compilation errors because base::WriteFileDescriptor.
    # Should be solved by using older libchrome or updating the package.
    # Perfectly fine package otherwise.
    'chromeos-base/ureadahead-diff',

    # Reason: REQUIRED_USE="kvm_guest" fails build.
    # Perfectly fine package otherwise.
    'chromeos-base/vm_guest_tools',

    # Reason: Fails build because it cannot find src/aosp/external/perfetto.
    # It's a go package that pretends to be an actual package. Should be
    # properly ignored.
    'dev-go/perfetto-protos',

    # Reason: Requires media-libs/intel-ipu6-camera-bins which is missing.
    # Perfectly fine package otherwise.
    'media-libs/cros-camera-hal-intel-ipu6',

    # Reason: Compilation errors due to some script.
    # Perfectly fine package otherwise.
    'media-libs/cros-camera-libjda_test',
}

# Set of packages that are not currently supported when building with tests.
TEMPORARY_UNSUPPORTED_PACKAGES_WITH_TESTS = {
    'chromeos-base/screen-capture-utils',
    'chromeos-base/update_engine',
    'chromeos-base/mtpd',
}
