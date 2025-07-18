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

PREBUILT_REMOTE_IMAGE=ghcr.io/kubeflow/kfp-api-generator:1.2

# Set USE_FIND_LINKS=true to use local sdk and pipeline_spec for buildling python builds rather than pulling from pypi
# applies to `make python` and make pythone-dev` commands.
# e.g. make python USE_FIND_LINKS=true FIND_LINKS_PATH="path1 path2 ..."
FIND_LINKS_PATH := /go/src/github.com/kubeflow/pipelines/sdk/python/dist /go/src/github.com/kubeflow/pipelines/api/v2alpha1/python/dist

.PHONY: all
all: golang python

.PHONY: clean
clean: clean-go clean-python

# Generate proto packages using a pre-built api-generator image.
.PHONY: golang
golang: proto/*.proto
	docker run --interactive --rm \
		--user $$(id -u):$$(id -g) \
		-v "$$(pwd)/..":"/go/src/github.com/kubeflow/pipelines":z \
		$(PREBUILT_REMOTE_IMAGE)  \
		sh -c 'cd /go/src/github.com/kubeflow/pipelines/kubernetes_platform && make generate-go-in-container'

# Delete all generated proto go packages.
.PHONY: clean-go
clean-go:
	rm -rf kubernetes_platform/go

.PHONY: fetch-protos
fetch-protos:
		make -C ../api fetch-protos

# Build Python package using pre-built image
.PHONY: python
python: proto/kubernetes_executor_config.proto fetch-protos
	docker run --interactive --rm \
		--user $$(id -u):$$(id -g) \
		-e HOME=/tmp \
		$(if $(USE_FIND_LINKS),-e PIP_FIND_LINKS="$(FIND_LINKS_PATH)") \
		-v "$$(pwd)/..":"/go/src/github.com/kubeflow/pipelines":z \
		$(PREBUILT_REMOTE_IMAGE) \
		sh -c 'cd /go/src/github.com/kubeflow/pipelines/kubernetes_platform/python && \
		    python3 -m pip install --user --break-system-packages -r requirements.txt && \
		    python3 generate_proto.py && \
		    python3 setup.py sdist bdist_wheel --dist-dir ./dist'

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
