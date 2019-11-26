#!/bin/bash -ex
# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

arch=$(uname -m)
if [[ $arch == *"aarch64"* ]]; then
  # Add packages required to run qemu-kvm in linux-arm64 docker image.
  arch_packages="libatomic1 libvirt-bin llvm virtinst"
  arch_suffix="_arm64"
else
  arch_packages=""
  arch_suffix=""
fi

date=$(/bin/date +"%Y-%m-%d_%H-%M")
par_dir="$(dirname "${0}")"

echo "Building for swarm_docker..."
/usr/bin/docker build \
  --no-cache=true \
  --pull \
  --build-arg "ARCH_PACKAGES=${arch_packages}" \
  -t "swarm_docker${arch_suffix}:${date}" \
  -t "swarm_docker${arch_suffix}:latest" \
  "${par_dir}"

exit 0
