# Copyright 2021 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Makefile to generate KFP api clients from proto.

IMAGE_TAG=kfp-api-generator
# Remote image for publishing (used by CI)
REMOTE_IMAGE=ghcr.io/kubeflow/kfp-api-generator
# Assume the latest API version by default.
API_VERSION ?= v2beta1

# Pre-built images for faster development (default: enabled for speed)
# Set USE_PREBUILT_IMAGE=false when:
# - Dockerfile dependencies have been updated
# - CI validation fails due to tool version mismatches
# - Working on API generation toolchain changes
USE_PREBUILT_IMAGE ?= true
# Validate USE_PREBUILT_IMAGE value
ifneq ($(USE_PREBUILT_IMAGE),true)
ifneq ($(USE_PREBUILT_IMAGE),false)
$(error USE_PREBUILT_IMAGE must be 'true' or 'false', got '$(USE_PREBUILT_IMAGE)')
endif
endif
PREBUILT_REMOTE_IMAGE=ghcr.io/kubeflow/kfp-api-generator:master
RELEASE_IMAGE=ghcr.io/kubeflow/kfp-release:master
CONTAINER_ENGINE ?= docker

# Generate clients using either pre-built image (fast) or source build (current tools).
# Use USE_PREBUILT_IMAGE=false when pre-built image may have outdated dependencies.
.PHONY: generate
generate: fetch-dependencies hack/generator.sh $(API_VERSION)/*.proto
ifeq ($(USE_PREBUILT_IMAGE),true)
	${CONTAINER_ENGINE} run --interactive --rm \
		-e API_VERSION=$(API_VERSION) \
		--user $$(id -u):$$(id -g) \
		--mount type=bind,source="$$(pwd)/../..",target=/go/src/github.com/kubeflow/pipelines \
		$(PREBUILT_REMOTE_IMAGE) /go/src/github.com/kubeflow/pipelines/backend/api/hack/generator.sh
else
	$(MAKE) generate-from-source
endif

# Generate clients by building api-generator image from source.
# This ensures generated files always use current tool versions.
.PHONY: generate-from-source
generate-from-source: fetch-dependencies .image-built hack/generator.sh $(API_VERSION)/*.proto
	${CONTAINER_ENGINE} run --interactive --rm \
		-e API_VERSION=$(API_VERSION) \
		--user $$(id -u):$$(id -g) \
		--mount type=bind,source="$$(pwd)/../..",target=/go/src/github.com/kubeflow/pipelines \
		$(IMAGE_TAG) /go/src/github.com/kubeflow/pipelines/backend/api/hack/generator.sh

# Generate KFP server API package using either pre-built image (fast) or source build (current tools).
# Use USE_PREBUILT_IMAGE=false when pre-built image may have outdated dependencies.
.PHONY: generate-kfp-server-api-package
generate-kfp-server-api-package:
ifeq ($(USE_PREBUILT_IMAGE),true)
	${CONTAINER_ENGINE} run --interactive --rm \
		-e API_VERSION=$(API_VERSION) \
		--user $$(id -u):$$(id -g) \
		--mount type=bind,source="$$(pwd)/../..",target=/go/src/github.com/kubeflow/pipelines \
		$(RELEASE_IMAGE) /go/src/github.com/kubeflow/pipelines/backend/api/build_kfp_server_api_python_package.sh
else
	$(MAKE) generate-kfp-server-api-package-from-source
endif

# Generate KFP server API package by building api-generator image from source.
# The API generator image now includes Java for OpenAPI generator.
.PHONY: generate-kfp-server-api-package-from-source
generate-kfp-server-api-package-from-source: .image-built
	${CONTAINER_ENGINE} run --interactive --rm \
		-e API_VERSION=$(API_VERSION) \
		--user $$(id -u):$$(id -g) \
		--mount type=bind,source="$$(pwd)/../..",target=/go/src/github.com/kubeflow/pipelines \
		$(IMAGE_TAG) /go/src/github.com/kubeflow/pipelines/backend/api/build_kfp_server_api_python_package.sh


# Fetch dependency proto
.PHONY: fetch-dependencies
fetch-dependencies: v2beta1/google/rpc/status.proto

v2beta1/google/rpc/status.proto:
	@mkdir -p v2beta1/google/rpc
	@# Authenticate with GITHUB_TOKEN when set to avoid anonymous per-IP 429
	@# rate limits on raw.githubusercontent.com. The bearer header is passed
	@# through a temporary WGETRC file, never on the command line. If the
	@# authenticated fetch fails it falls back to an anonymous fetch, so
	@# behavior is never worse than without a token.
	@wget_cmd="wget --tries=3 --waitretry=10 --retry-on-http-error=429,500,502,503 -O v2beta1/google/rpc/status.proto https://raw.githubusercontent.com/googleapis/googleapis/047d3a8ac7f75383855df0166144f891d7af08d9/google/rpc/status.proto"; \
	if [ -n "$$GITHUB_TOKEN" ]; then \
		wgetrc=$$(mktemp "$${TMPDIR:-/tmp}/kfp-proto-wgetrc.XXXXXX"); \
		printf 'header = Authorization: Bearer %s\n' "$$GITHUB_TOKEN" > "$$wgetrc"; \
		WGETRC=$$wgetrc $$wget_cmd || $$wget_cmd; \
		status=$$?; \
		rm -f "$$wgetrc"; \
	else \
		$$wget_cmd; \
		status=$$?; \
	fi; \
	exit $$status

# Legacy target - always builds from source for backward compatibility.
.PHONY: generate-from-scratch
generate-from-scratch: generate-from-source

# Build a local api-generator image.
.PHONY: image
image: .image-built

# Push api-generator image remotely.
.PHONY: push
push: image
	${CONTAINER_ENGINE} tag $(IMAGE_TAG) $(REMOTE_IMAGE)
	${CONTAINER_ENGINE} push $(REMOTE_IMAGE)

# .image-built is a local token file to help Make determine the latest successful build.
# Uses Docker layer caching for improved performance.
.image-built: Dockerfile
	DOCKER_BUILDKIT=1 ${CONTAINER_ENGINE} build ../.. -t $(IMAGE_TAG) -f Dockerfile \
		--cache-from $(IMAGE_TAG):latest
	touch .image-built
