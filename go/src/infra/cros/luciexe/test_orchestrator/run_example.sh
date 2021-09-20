#!/usr/bin/env bash
#
# This script runs the Test Orchestrator on an example Build proto input. It
# also sets up required environment, such as the Logdog Butler.

set -e

readonly script_dir="$(dirname "$(realpath -e "${BASH_SOURCE[0]}")")"

cd "${script_dir}"

readonly base_repo_path=$(realpath "${script_dir}/../../../../../..")
readonly base_proto_path="${base_repo_path}/go/src"
readonly build_proto_path="${base_proto_path}/go.chromium.org/luci/buildbucket/proto/build.proto"

readonly work_dir=$(mktemp -d)
trap 'rm -rf ${work_dir}' EXIT

readonly proto_output_path="${work_dir}/build.pb"

echo "Writing sample Build proto to ${build_proto_path}..."
protoc --encode buildbucket.v2.Build \
    -I"${base_proto_path}" -I"${base_repo_path}/appengine/monorail" \
    "${build_proto_path}" \
< example_req.textpb > "${proto_output_path}"

echo "Getting logdog_butler..."
go get go.chromium.org/luci/logdog/client/cmd/logdog_butler

echo "Running Test Orchestrator..."
PATH=$(go env GOBIN):${PATH} logdog_butler -project test-project -output log \
    run -forward-stdin \
    go run luciexe.go --strict-input < "${proto_output_path}"