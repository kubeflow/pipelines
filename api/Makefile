# Copyright 2022 The Kubeflow Authors
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

# Contact one of chensun, HumairAK if this remote image needs an update.
PREBUILT_REMOTE_IMAGE=ghcr.io/kubeflow/kfp-api-generator:1.2

.PHONY: all
all: golang python

.PHONY: clean
clean: clean-go clean-python

# Generate proto packages using a pre-built api-generator image.
.PHONY: golang
golang: v2alpha1/*.proto
	docker run --interactive --rm \
		--user $$(id -u):$$(id -g) \
		--mount type=bind,source="$$(pwd)/..",target=/go/src/github.com/kubeflow/pipelines \
		$(PREBUILT_REMOTE_IMAGE)  \
		sh -c 'cd /go/src/github.com/kubeflow/pipelines/api && make generate'

# Delete all generated proto go packages.
.PHONY: clean-go
clean-go:
	rm -rf v2alpha1/go
	rm -f v2alpha1/google/rpc/status.proto

# Build Python package using pre-built image
.PHONY: python
python: python fetch-protos
	docker run --interactive --rm \
		--user $$(id -u):$$(id -g) \
		-e HOME=/tmp \
		-v "$$(pwd)/..":"/go/src/github.com/kubeflow/pipelines":z \
		$(PREBUILT_REMOTE_IMAGE) \
		sh -c 'cd /go/src/github.com/kubeflow/pipelines/api/v2alpha1/python && \
		    python3 -m pip install --user --break-system-packages -r requirements.txt && \
		    python3 generate_proto.py && \
		    python3 setup.py sdist bdist_wheel --dist-dir ./dist'

# Build and locally install Python package using editable mode for development.
.PHONY: python-dev
python-dev: v2alpha1/pipeline_spec.proto fetch-protos
	python3 v2alpha1/python/generate_proto.py && cd v2alpha1/python && pip install -e .

# Delete all generated Python packages
.PHONY: clean-python
clean-python:
	rm -rf v2alpha1/python/build
	rm -rf v2alpha1/python/dist
	rm -rf v2alpha1/python/kfp_pipeline_spec.egg-info
	rm -rf v2alpha1/google
	rm -f v2alpha1/python/kfp/pipeline_spec/pipeline_spec_pb2.py
	rm -f v2alpha1/google/rpc/status.proto

###########################################
# The following are IMPLEMENTATION DETAILS.
###########################################

# Generates proto packages locally, this should only be called:
# * during development
# * inside the prebuilt docker container
.PHONY: generate
generate: go_pipelinespec go_cachekey

go_pipelinespec: v2alpha1/pipeline_spec.proto fetch-protos
	mkdir -p v2alpha1/go/pipelinespec
	cd v2alpha1 && protoc -I=. \
		--go_out=go/pipelinespec \
		--go_opt=paths=source_relative \
		pipeline_spec.proto

go_cachekey: v2alpha1/pipeline_spec.proto v2alpha1/cache_key.proto
	mkdir -p v2alpha1/go/cachekey
	cd v2alpha1 && protoc -I=. \
		--go_out=go/cachekey \
		--go_opt=paths=source_relative \
		cache_key.proto

#########################################
# The following are dependencies
# Required for compiling the proto files
#########################################

GOOGLEAPIS_COMMIT ?= fecd7d35f46753b45bf4519f6342495a181740c9
PROTOBUF_TAG ?= v26.0
PROTO_DST_DIR := v2alpha1/google
TMP_PROTOBUF_DIR := /tmp/protobuf-src

.PHONY: fetch-protos fetch-googleapis fetch-protobuf protoc-gen-go clean-protobuf-tmp

fetch-protos: fetch-googleapis fetch-protobuf

fetch-googleapis:
	@echo "Downloading google/rpc/status.proto from googleapis@$(GOOGLEAPIS_COMMIT)..."
	mkdir -p $(PROTO_DST_DIR)/rpc
	wget -qO $(PROTO_DST_DIR)/rpc/status.proto https://raw.githubusercontent.com/googleapis/googleapis/$(GOOGLEAPIS_COMMIT)/google/rpc/status.proto

fetch-protobuf: clean-protobuf-tmp
	@git clone --depth 1 --branch $(PROTOBUF_TAG) https://github.com/protocolbuffers/protobuf.git $(TMP_PROTOBUF_DIR)
	@mkdir -p $(PROTO_DST_DIR)/protobuf
	@cp $(TMP_PROTOBUF_DIR)/src/google/protobuf/*.proto $(PROTO_DST_DIR)/protobuf/

clean-protobuf-tmp:
	@rm -rf $(TMP_PROTOBUF_DIR)

protoc-gen-go:
	go install google.golang.org/protobuf/cmd/protoc-gen-go
