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

# Contact one of Bobgy, capri-xiyue or zijianjoy if this remote image needs an update.
PREBUILT_REMOTE_IMAGE=gcr.io/ml-pipeline-test/api-generator:latest

# Generate proto packages using a pre-built api-generator image.
.PHONY: all
all: v2alpha1/*.proto
	docker run --interactive --rm \
		--user $$(id -u):$$(id -g) \
		--mount type=bind,source="$$(pwd)/..",target=/go/src/github.com/kubeflow/pipelines \
		$(PREBUILT_REMOTE_IMAGE)  \
		sh -c 'cd /go/src/github.com/kubeflow/pipelines/api && make generate'

# Delete all generated proto go packages.
.PHONY: clean
clean:
	rm -rf v2alpha1/go

##########################
# The following are IMPLEMENTATION DETAILS.
##########################

# Generates proto packages locally, this should only be called:
# * during development
# * inside the prebuilt docker container
.PHONY: generate
generate: go_pipelinespec go_cachekey

go_pipelinespec: v2alpha1/pipeline_spec.proto
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

# protoc-gen-go is already installed in api-generator image
.PHONY: protoc-gen-go
protoc-gen-go:
	go install google.golang.org/protobuf/cmd/protoc-gen-go
