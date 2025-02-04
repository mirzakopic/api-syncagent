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

# build the image(s)
export IMAGE_TAG=local

echo "Building container images…"
# ARCHITECTURES=amd64 DRY_RUN=yes ./hack/ci/build-image.sh

# start docker so we can run kind
# start-docker.sh

# create a local kind cluster
KIND_CLUSTER_NAME=e2e

# echo "Preloading the kindest/node image…"
# docker load --input /kindest.tar

export KUBECONFIG=$(mktemp)
export KUBECONFIG=e2e.kubeconfig
echo "Creating kind cluster $KIND_CLUSTER_NAME…"
kind create cluster --name "$KIND_CLUSTER_NAME"
chmod 600 "$KUBECONFIG"

# load the agent image into the kind cluster
image="ghcr.io/kcp-dev/api-syncagent:$IMAGE_TAG"
archive=agent.tar

echo "Loading api-syncagent image into kind…"
buildah manifest push --all "$image" "oci-archive:$archive:$image"
kind load image-archive "$archive" --name "$KIND_CLUSTER_NAME"

# deploy cert-manager
echo "Installing cert-manager…"

helm repo add jetstack https://charts.jetstack.io
helm repo update

kubectl apply --filename https://github.com/cert-manager/cert-manager/releases/download/v1.17.0/cert-manager.crds.yaml
helm install \
  --wait \
  --namespace cert-manager \
  --create-namespace \
  --version v1.17.0 \
  cert-manager jetstack/cert-manager

# deploy a kcp which will live for the entire runtime of the e2e tests and be shared among all subtests
echo "Installing kcp into kind…"

helm repo add kcp https://kcp-dev.github.io/helm-charts
helm repo update

helm install \
  --wait \
  --namespace kcp \
  --create-namespace \
  --values hack/ci/testdata/kcp-kind-values.yaml \
  kcp-e2e kcp/kcp

# time to run the tests
echo "Running e2e tests…"
(set -x; go test -tags e2e -timeout 2h -v ./test/e2e/...)

echo "Done. :-)"
