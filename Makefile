# Copyright 2024 The Kubermatic Kubernetes Platform contributors.
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

export CGO_ENABLED ?= 0
export GOFLAGS ?= -mod=readonly -trimpath
export GO111MODULE = on
CMD ?= $(filter-out OWNERS, $(notdir $(wildcard ./cmd/*)))
GOBUILDFLAGS ?= -v
GIT_HEAD ?= $(shell git log -1 --format=%H)
GIT_VERSION = $(shell git describe --tags --always)
LDFLAGS += -extldflags '-static' \
  -X k8c.io/servlet/internal/version.gitVersion=$(GIT_VERSION) \
  -X k8c.io/servlet/internal/version.gitHead=$(GIT_HEAD)
BUILD_DEST ?= _build
GOTOOLFLAGS ?= $(GOBUILDFLAGS) -ldflags '-w $(LDFLAGS)' $(GOTOOLFLAGS_EXTRA)

.PHONY: all
all: build test

.PHONY: build
build: $(CMD)

.PHONY: $(CMD)
$(CMD): %: $(BUILD_DEST)/%

$(BUILD_DEST)/%: cmd/%
	go build $(GOTOOLFLAGS) -o $@ ./cmd/$*

.PHONY: test
test:
	./hack/run-tests.sh

.PHONY: codegen
codegen:
	hack/update-codegen-crds.sh
	hack/update-codegen-sdk.sh

.PHONY: build-tests
build-tests:
	go test -run nope ./...

.PHONY: clean
clean:
	rm -rf $(BUILD_DEST)
	@echo "Cleaned $(BUILD_DEST)"

.PHONY: lint
lint:
	golangci-lint run \
		--verbose \
		--print-resources-usage \
		./...

.PHONY: verify
verify:
	./hack/verify-boilerplate.sh
	./hack/verify-import-order.sh
	./hack/verify-licenses.sh
