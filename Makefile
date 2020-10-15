# Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

IMAGE_REPOSITORY    := eu.gcr.io/gardener-project/gardener/machine-controller-manager-provider-kubevirt
IMAGE_TAG           := $(shell cat VERSION)
CONTROL_NAMESPACE  := default
CONTROL_KUBECONFIG := dev/target-kubeconfig.yaml
TARGET_KUBECONFIG  := dev/target-kubeconfig.yaml

#########################################
# Rules for starting machine-controller locally
#########################################

.PHONY: start
start:
	@GO111MODULE=on go run \
		cmd/machine-controller/main.go \
		--control-kubeconfig=$(CONTROL_KUBECONFIG) \
		--target-kubeconfig=$(TARGET_KUBECONFIG) \
		--namespace=$(CONTROL_NAMESPACE) \
		--machine-creation-timeout=20m \
		--machine-drain-timeout=5m \
		--machine-health-timeout=10m \
		--machine-pv-detach-timeout=2m \
		--machine-safety-apiserver-statuscheck-timeout=30s \
		--machine-safety-apiserver-statuscheck-period=1m \
		--machine-safety-orphan-vms-period=30m \
		--v=3

#########################################
# Rules for re-vendoring
#########################################

.PHONY: revendor
revendor:
	@env GO111MODULE=on go mod vendor -v
	@env GO111MODULE=on go mod tidy -v

#########################################
# Rules for code generation
#########################################

.PHONY: generate
generate:
	@env GO111MODULE=on go generate -mod=vendor ./pkg/...

#########################################
# Rules for testing
#########################################

.PHONY: verify
verify: check test

.PHONY: test
test:
	@.ci/test

.PHONY: check
check:
	@.ci/check

#########################################
# Rules for build/release
#########################################

.PHONY: release
release: build-local build docker-image docker-login docker-push rename-binaries

.PHONY: build-local
build-local:
	@env LOCAL_BUILD=1 .ci/build

.PHONY: build
build:
	@.ci/build

.PHONY: docker-image
docker-image:
	@docker build -t $(IMAGE_REPOSITORY):$(IMAGE_TAG) .

.PHONY: docker-login
docker-login:
	@gcloud auth activate-service-account --key-file .kube-secrets/gcr/gcr-readwrite.json

.PHONY: docker-push
docker-push:
	@if ! docker images $(IMAGE_REPOSITORY) | awk '{ print $$2 }' | grep -q -F $(IMAGE_TAG); then echo "$(IMAGE_REPOSITORY) version $(IMAGE_TAG) is not yet built. Please run 'make docker-image'"; false; fi
	@gcloud docker -- push $(IMAGE_REPOSITORY):$(IMAGE_TAG)

.PHONY: rename-binaries
rename-binaries:
	@if [[ -f bin/machine-controller ]]; then cp bin/machine-controller machine-controller-darwin-amd64; fi
	@if [[ -f bin/rel/machine-controller ]]; then cp bin/rel/machine-controller machine-controller-linux-amd64; fi

.PHONY: clean
clean:
	@rm -rf bin/
	@rm -f *linux-amd64
	@rm -f *darwin-amd64
