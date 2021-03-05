#!/bin/sh
# Copyright 2021 The Chromium OS Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
# This is not a real version file; it exists only for testing purposes.
#############################################################################
# SET VERSION NUMBERS
#############################################################################
if [ -z "${FLAGS_version}" ]; then
  # Release Build number.
  # Increment by 1 for every release build.
  CHROMEOS_BUILD=13781
  # Release Branch number.
  # Increment by 1 for every release build on a branch.
  # Reset to 0 when increasing release build number.
  CHROMEOS_BRANCH=0
  # Patch number.
  # Increment by 1 in case a non-scheduled branch release build is necessary.
  # Reset to 0 when increasing branch number.
  CHROMEOS_PATCH=1
  # Official builds must set CHROMEOS_OFFICIAL=1.
  if [ ${CHROMEOS_OFFICIAL:-0} -ne 1 ]; then
    # For developer builds, overwrite CHROMEOS_PATCH with a date string
    # for use by auto-updater.
    CHROMEOS_PATCH=$(date +%Y_%m_%d_%H%M)
  fi
  # Version string. Not indented to appease bash.
  CHROMEOS_VERSION_STRING=\
"${CHROMEOS_BUILD}.${CHROMEOS_BRANCH}.${CHROMEOS_PATCH}"
else
  CHROMEOS_BUILD=$(echo "${FLAGS_version}" | cut -f 1 -d ".")
  CHROMEOS_BRANCH=$(echo "${FLAGS_version}" | cut -f 2 -d ".")
  CHROMEOS_PATCH=$(echo "${FLAGS_version}" | cut -f 3 -d ".")
  CHROMEOS_VERSION_STRING="${FLAGS_version}"
fi
# Major version for Chrome.
CHROME_BRANCH=90
# Set CHROME values (Used for releases) to pass to chromeos-chrome-bin ebuild
# URL to chrome archive
CHROME_BASE=
# Set CHROME_VERSION from incoming value or NULL and let ebuild default.
: "${CHROME_VERSION:=}"
# Print (and remember) version info.  We do each one by hand because there might
# be more/other vars in the env already that start with CHROME_ or CHROMEOS_.
echo "Chromium OS version information:"
(
# Subshell to hide the show_vars definition.
show_vars() {
  local v
  for v in "$@"; do
    eval echo \""    ${v}=\${${v}}"\"
  done
}
show_vars \
  CHROME_BASE \
  CHROME_BRANCH \
  CHROME_VERSION \
  CHROMEOS_BUILD \
  CHROMEOS_BRANCH \
  CHROMEOS_PATCH \
  CHROMEOS_VERSION_STRING
)