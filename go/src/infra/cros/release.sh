#!/bin/bash
# Copyright 2022 The Chromium OS Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Releases a CrOS golang bundle to prod.
# Will release either the CI or CTP Golang binaries as defined by
# the `chromeos/infra/ci-uprev-prod` and `chromeos/infra/ctp-uprev-prod`
# builders. These builders can be invoked directly; the primary purpose of this
# script is to show the pending changes.

set -eu

no_changes="No changes pending."
cros_golang_root="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
recipes_bundle="infra/recipe_bundles/chromium.googlesource.com/chromiumos/infra/recipes"

if ! type jq >/dev/null; then
  echo "Please install jq, on debian:"
  printf "\tsudo apt install jq\n"
  exit 1
fi

usage="Release CrOS Golang packages (CI or CTP).

Usage: ${0} [options] {ci|ctp}

Options:
  -f,--force      bypasses the prompt
  -v,--verbose    print all changes in go/, not just those in go/src/infra/cros
  -h, --help      This help output."


function urlencode() {
  # The sed magic strips color codes.
  echo "$1" | sed 's/\x1b\[[0-9;]*m//g' | jq -sRr @uri
}

function check_bb_auth() {
  if ! bb auth-info > /dev/null; then
    bb auth-login
  fi
}

# Get the git revision associated with a cipd version (dereference it).
cipd_version_to_githash() {
  cipd describe -json-output /proc/self/fd/2 -version "$2" "$1" 2>&1 > /dev/null |
    jq -r '.result.tags|map(.tag|select(startswith("git_revision:")))[0]|sub(".*:";"")'
}

# Print CrOS golang commits pending release to production.
cros_golang_pending() {
  log_fmt="%C(bold blue)%h %C(bold green)[%al]%C(auto)%d %C(reset)%s"
  cmd=(git -C "${cros_golang_root}" log --color --graph --decorate \
      --pretty=format:"${log_fmt}" "${latest_prod_sha}".."${earliest_staging_sha}")
  if  [[ "$verbose" == "no" ]]; then
    cmd+=(-- "${cros_golang_root}")
  else
    cmd+=(-- "${cros_golang_root}"/../../..)
  fi
  changes=$("${cmd[@]}")

  if [ -z "${changes}" ]; then
      echo "${no_changes}"
  else
      while IFS= read -r line; do
        echo -e "  ${line}"
      done <<< "${changes}"
  fi
}

# "Main" function.
prompt="yes"
verbose="no"
which_golang=""
while [[ $# -ne 0 ]]; do
  case $1 in
    -f|--force) prompt="no";;
    -v|--verbose) verbose="yes";;
    -h|--help)
      exec printf '%b\n' "${usage}"
      ;;
    -*)
      echo "invalid option $1" >/dev/stderr
      exec printf '\nUsage:\n%b\n' "${usage}"
      exit 1
      ;;
    *)
      which_golang="$1"
      ;;
  esac
  shift
done

if [[ -z "$which_golang" ]]; then
  echo "must select a set of binaries to release (ci or ctp)"
  echo
  exec printf '%b\n' "${usage}"
  exit 1
fi
uprev_builder=""
staging_builder_name=""
if [[ "$which_golang" == "ci" ]] || [[ "$which_golang" == "CI" ]]; then
  which_golang="CI"
  uprev_builder="chromeos/infra/ci-uprev-prod"
  staging_builder_name="ci-uprev-staging"
elif [[ "$which_golang" == "ctp" ]] || [[ "$which_golang" == "CTP" ]]; then
  which_golang="CTP"
  uprev_builder="chromeos/infra/ctp-uprev-prod"
  staging_builder_name="ctp-uprev-staging"
else
  echo "must select a valid set of binaries to release (ci or ctp)"
  echo
  exec printf '%b\n' "${usage}"
  exit 1
fi

# Update to remote.
git -C "${cros_golang_root}" remote update > /dev/null

output=$(bb ls ${uprev_builder} -n 1 -p)
packages=$(grep package_name <<< $output | awk '{printf "%s\n", substr($NF, 2, length($NF)-3)}')
packages=($packages)

# We're releasing multiple packages whose staging and prod labels may be at
# different SHAs. Find the most includive range (earliest start point and
# latest end point).
earliest_staging_sha=""
earliest_staging_sha_time=0
latest_prod_sha=""
latest_prod_sha_time=0
for package in "${packages[@]}"
do
  if [[ $package == "$recipes_bundle" ]]; then
    continue
  fi
  sha=$(cipd_version_to_githash $package "staging")
  sha_time=$(git show -s --format=%ct $sha)
  if (( $earliest_staging_sha_time == 0 )) || (( earliest_staging_sha_time < sha_time )); then
    earliest_staging_sha_time=$sha_time
    earliest_staging_sha=$sha
  fi
  sha=$(cipd_version_to_githash $package "prod")
  sha_time=$(git show -s --format=%ct $sha)
  if (( $latest_prod_sha_time == 0 )) || (( latest_prod_sha_time > sha_time )); then
    latest_prod_sha_time=$sha_time
    latest_prod_sha=$sha
  fi
done

echo "CIPD versions for various packages can be found here: https://chrome-infra-packages.appspot.com/p/chromiumos/infra"
echo

echo "=== Checking for pending changes ==="
printf "Here are the changes from the provided (or default main) environment:\n"
pending=$(cros_golang_pending)
echo "${pending}"
echo
echo "Is your change not listed? It might not have been picked up by the staging builder yet (https://ci.chromium.org/p/chromeos/builders/infra/$staging_builder_name)."
echo

if [[ $pending == "$no_changes" ]]; then
    echo "No changes pending. Exiting early."
    exit 0
fi

# TODO(b/217943800): Staging health check.

if [[ "${prompt}" == "yes" ]]; then
 read -rp "Deploy to prod? (y/N): " answer
 if [[ "${answer^^}" != "Y" ]]; then
   exit 0
 fi
fi

check_bb_auth
output=$(bb add "${uprev_builder}")
uprev_build=$(echo $output | head -n1 | awk '{printf $1}')
packages=$(grep package_name <<< $output | awk '{printf "%s\n", substr($NF, 2, length($NF)-3)}')

email_subject="ChromeOS ${which_golang} Golang Release - $(TZ='America/Los_Angeles' date)"
email_message="We've yeeted ${which_golang} Golang to prod!

Uprev build: ${uprev_build}

Deployed packages:
${packages}

Here is a summary of the changes:

${pending}"

email_link="https://mail.google.com/mail/?view=cm&fs=1&bcc=chromeos-infra-releases@google.com&to=chromeos-continuous-integration-team@google.com&su=$(urlencode "${email_subject}")&body=$(urlencode "${email_message}")"

echo
echo "Please click this link and send an email to chromeos-infra-releases!"
echo
echo "${email_link}"