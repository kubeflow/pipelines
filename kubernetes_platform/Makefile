# Copyright 2023 The Kubeflow Authors
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
IMAGE_TAG=kfp-api-generator

# Set USE_FIND_LINKS=true to use local sdk and pipeline_spec for buildling python builds rather than pulling from pypi
# applies to `make python` and make pythone-dev` commands.
# e.g. make python USE_FIND_LINKS=true FIND_LINKS_PATH="path1 path2 ..."
FIND_LINKS_PATH := /go/src/github.com/kubeflow/pipelines/sdk/python/dist /go/src/github.com/kubeflow/pipelines/api/v2alpha1/python/dist /go/src/github.com/kubeflow/pipelines/backend/api/v2beta1/python_http_client/dist

.PHONY: all
all: golang python

.PHONY: clean
clean: clean-go clean-python

# Generate proto packages using either pre-built image (fast) or source build (accurate).
# Use USE_PREBUILT_IMAGE=false to build from source for guaranteed accuracy.
.PHONY: golang
golang: proto/*.proto
ifeq ($(USE_PREBUILT_IMAGE),true)
	docker run --interactive --rm \
		--user $$(id -u):$$(id -g) \
		-v "$$(pwd)/..":"/go/src/github.com/kubeflow/pipelines":z \
		$(PREBUILT_REMOTE_IMAGE)  \
		sh -c 'cd /go/src/github.com/kubeflow/pipelines/kubernetes_platform && make generate-go-in-container'
else
	$(MAKE) golang-from-source
endif

# Generate proto packages by building api-generator image from source.
.PHONY: golang-from-source
golang-from-source: .api-generator-built proto/*.proto
	docker run --interactive --rm \
		--user $$(id -u):$$(id -g) \
		-v "$$(pwd)/..":"/go/src/github.com/kubeflow/pipelines":z \
		$(IMAGE_TAG)  \
		sh -c 'cd /go/src/github.com/kubeflow/pipelines/kubernetes_platform && make generate-go-in-container'

# Delete all generated proto go packages.
.PHONY: clean-go
clean-go:
	rm -rf kubernetes_platform/go

.PHONY: fetch-protos
fetch-protos:
		make -C ../api fetch-protos

# Build Python package using either pre-built image (fast) or source build (accurate).
# Use USE_PREBUILT_IMAGE=false to build from source for guaranteed accuracy.
.PHONY: python
python: proto/kubernetes_executor_config.proto fetch-protos
ifeq ($(USE_PREBUILT_IMAGE),true)
	docker run --interactive --rm \
		--user $$(id -u):$$(id -g) \
		-e HOME=/tmp \
		$(if $(USE_FIND_LINKS),-e PIP_FIND_LINKS="$(FIND_LINKS_PATH)") \
		-v "$$(pwd)/..":"/go/src/github.com/kubeflow/pipelines":z \
		$(PREBUILT_REMOTE_IMAGE) \
		sh -c 'cd /go/src/github.com/kubeflow/pipelines/kubernetes_platform/python && \
		    python3 -m pip install --user --break-system-packages -r requirements.txt && \
		    python3 generate_proto.py && \
		    python3 setup.py sdist && pip wheel --no-deps dist/*.tar.gz -w dist'
else
	$(MAKE) python-from-source
endif

# Build Python package by building api-generator image from source.
.PHONY: python-from-source
python-from-source: .api-generator-built proto/kubernetes_executor_config.proto fetch-protos
	docker run --interactive --rm \
		--user $$(id -u):$$(id -g) \
		-e HOME=/tmp \
		$(if $(USE_FIND_LINKS),-e PIP_FIND_LINKS="$(FIND_LINKS_PATH)") \
		-v "$$(pwd)/..":"/go/src/github.com/kubeflow/pipelines":z \
		$(IMAGE_TAG) \
		sh -c 'cd /go/src/github.com/kubeflow/pipelines/kubernetes_platform/python && \
		    python3 -m pip install --user --break-system-packages -r requirements.txt && \
		    python3 generate_proto.py && \
		    python3 setup.py sdist && pip wheel --no-deps dist/*.tar.gz -w dist'

# Build and install in editable mode using pre-built image
.PHONY: python-dev
python-dev: proto/kubernetes_executor_config.proto fetch-protos
	python3 python/generate_proto.py && cd python && pip install -e .[dev]

# Delete all generated Python packages
.PHONY: clean-python
clean-python:
	rm -rf python/build
	rm -rf python/dist
	rm -rf python/kfp_kubernetes.egg-info
	rm -f python/kfp/kubernetes/kubernetes_executor_config_pb2.py

##########################
# IMPLEMENTATION DETAILS -- run inside of container for generating go
##########################

.PHONY: generate-go-in-container
generate-go-in-container:
	mkdir -p go/kubernetesplatform
	cd proto && protoc -I=. -I=../../api/v2alpha1 \
		--go_out=../go/kubernetesplatform \
		--go_opt=paths=source_relative \
		kubernetes_executor_config.proto

# Build the API generator image locally to ensure current tool versions
.api-generator-built: ../backend/api/Dockerfile
	DOCKER_BUILDKIT=1 docker build .. -t $(IMAGE_TAG) -f ../backend/api/Dockerfile \
		--cache-from $(IMAGE_TAG):latest
	touch .api-generator-built
