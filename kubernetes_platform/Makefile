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

PREBUILT_REMOTE_IMAGE=gcr.io/ml-pipeline-test/api-generator:latest

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


# Build and locally install Python package.
.PHONY: python
python: proto/kubernetes_executor_config.proto
	python3 python/generate_proto.py && cd python && python3 setup.py bdist_wheel

# Build and locally install Python package using editable mode for development.
.PHONY: python-dev
python-dev: proto/kubernetes_executor_config.proto
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
	cd proto && protoc -I=. \
		--go_out=../go/kubernetesplatform \
		--go_opt=paths=source_relative \
		kubernetes_executor_config.proto
