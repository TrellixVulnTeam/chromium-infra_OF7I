#!/bin/bash -ex
# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

arch=$(uname -m)
if [[ $arch == *"aarch64"* ]]; then
  # Add packages required to run femu in linux-arm64 docker image.
  # TODO(1162314): Remove llvm package after migration to new SDK symbolizer.
  arch_packages="google-perftools libatomic1 libgl1-mesa-glx libpcre2-16-0 llvm"
  kvm_packages="libvirt-daemon-system libvirt-clients virtinst"
  arch_suffix="_arm64"
  arch_build_options="--memory=8g"
else
  arch_packages=""
  kvm_packages=""
  arch_suffix=""
  arch_build_options=""
fi

date=$(/bin/date +"%Y-%m-%d_%H-%M")
par_dir="$(dirname "${0}")"

echo "Building for swarm_docker..."
/usr/bin/docker build \
  --no-cache=true \
  --pull \
  --build-arg "ARCH_PACKAGES=${arch_packages}" \
  --build-arg "KVM_PACKAGES=${kvm_packages}" \
  -t "swarm_docker${arch_suffix}:${date}" \
  -t "swarm_docker${arch_suffix}:latest" \
  ${arch_build_options} \
  "${par_dir}"

exit 0
