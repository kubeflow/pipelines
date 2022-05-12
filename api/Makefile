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

# Contact one of Bobgy, capri-xiyue or zijianjoy if this remote image needs an update.
PREBUILT_REMOTE_IMAGE=gcr.io/ml-pipeline-test/api-generator:latest

.PHONY: all
all: golang python

.PHONY: clean
clean: clean-go clean-python

# Generate proto packages using a pre-built api-generator image.
.PHONY: golang
golang: kfp_pipeline_spec/*.proto
	docker run --interactive --rm \
		--user $$(id -u):$$(id -g) \
		--mount type=bind,source="$$(pwd)/..",target=/go/src/github.com/kubeflow/pipelines \
		$(PREBUILT_REMOTE_IMAGE)  \
		sh -c 'cd /go/src/github.com/kubeflow/pipelines/api && make generate'

# Delete all generated proto go packages.
.PHONY: clean-go
clean-go:
	rm -rf kfp_pipeline_spec/go
	rm -f kfp_pipeline_spec/google/rpc/status.proto

# Generate Python package.
.PHONY: python
python: kfp_pipeline_spec/pipeline_spec.proto kfp_pipeline_spec/google/rpc/status.proto
	python3 kfp_pipeline_spec/python/generate_proto.py && cd kfp_pipeline_spec/python && python3 setup.py bdist_wheel

# Delete all generated Python packages
.PHONY: clean-python
clean-python:
	rm -rf kfp_pipeline_spec/python/build
	rm -rf kfp_pipeline_spec/python/dist
	rm -rf kfp_pipeline_spec/python/kfp_pipeline_spec.egg-info
	rm -f kfp_pipeline_spec/python/kfp/pipeline_spec/pipeline_spec_pb2.py
	rm -f kfp_pipeline_spec/google/rpc/status.proto

##########################
# The following are IMPLEMENTATION DETAILS.
##########################

# Generates proto packages locally, this should only be called:
# * during development
# * inside the prebuilt docker container
.PHONY: generate
generate: go_pipelinespec go_cachekey

go_pipelinespec: kfp_pipeline_spec/pipeline_spec.proto kfp_pipeline_spec/google/rpc/status.proto
	mkdir -p kfp_pipeline_spec/go/pipelinespec
	cd kfp_pipeline_spec && protoc -I=. \
		--go_out=go/pipelinespec \
		--go_opt=paths=source_relative \
		pipeline_spec.proto

go_cachekey: kfp_pipeline_spec/pipeline_spec.proto kfp_pipeline_spec/cache_key.proto
	mkdir -p kfp_pipeline_spec/go/cachekey
	cd kfp_pipeline_spec && protoc -I=. \
		--go_out=go/cachekey \
		--go_opt=paths=source_relative \
		cache_key.proto

# Fetch dependency proto
kfp_pipeline_spec/google/rpc/status.proto:
	mkdir -p kfp_pipeline_spec/google/rpc
	wget -O kfp_pipeline_spec/google/rpc/status.proto https://raw.githubusercontent.com/googleapis/googleapis/047d3a8ac7f75383855df0166144f891d7af08d9/google/rpc/status.proto

# protoc-gen-go is already installed in api-generator image
.PHONY: protoc-gen-go
protoc-gen-go:
	go install google.golang.org/protobuf/cmd/protoc-gen-go
