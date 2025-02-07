#!/usr/bin/env bash

# Copyright 2025 The KCP Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail
source hack/lib.sh

# build the agent, we will start it many times during the tests
echodate "Building the api-syncagent…"
make build

# get kube envtest binaries
echodate "Setting up Kube binaries…"
make _tools/setup-envtest
export KUBEBUILDER_ASSETS="$(_tools/setup-envtest use 1.31.0 --bin-dir _tools -p path)"
KUBEBUILDER_ASSETS="$(realpath "$KUBEBUILDER_ASSETS")"

# start a shared kcp process
make _tools/kcp

KCP_ROOT_DIRECTORY=.kcp.e2e
KCP_LOGFILE=kcp.e2e.log

echodate "Starting kcp…"
rm -rf "$KCP_ROOT_DIRECTORY" "$KCP_LOGFILE"
_tools/kcp start \
  --root-directory "$KCP_ROOT_DIRECTORY" 1>"$KCP_LOGFILE" 2>&1 &

stop_kcp() {
  echodate "Stopping kcp processes…"
  pkill -e kcp
}
append_trap stop_kcp EXIT

# Wait for kcp to be ready; this env name is also hardcoded in the Go tests.
export KCP_KUBECONFIG="$KCP_ROOT_DIRECTORY/admin.kubeconfig"

if ! retry_linear 3 20 kubectl --kubeconfig "$KCP_KUBECONFIG" get logicalcluster; then
  echodate "kcp never became ready."
  exit 1
fi

# time to run the tests
echodate "Running e2e tests…"
(set -x; go test -tags e2e -timeout 2h -v ./test/e2e/...)

echodate "Done. :-)"
