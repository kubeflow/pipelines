# Copyright 2021 The Kubeflow Authors
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

# we use some bash builtin features like $RANDOM
SHELL = /bin/bash

# create an .env file to override dev vars locally
-include .env
DEV_IMAGE_PREFIX?=gcr.io/ml-pipeline-test/dev/kfp-
REPO_ROOT=..
GOOS_VALUE?=linux
GOARCH_VALUE?=amd64
# override this via env to increase verbosity of go-licenses, example:
# GO_LICENSES_FLAGS=-v4 make license-launcher
GO_LICENSES_FLAGS?=
GO_FLAGS=GO111MODULE=on CGO_ENABLED=0 GOOS=${GOOS_VALUE} GOARCH=${GOARCH_VALUE}
GO_BUILD_FLAGS=-tags netgo -ldflags '-extldflags "-static"'
KO_VERSION?=v0.8.3
MLMD_VERSION=$$(cat ../third_party/ml-metadata/VERSION)

###### Common targets ######

# make test runs all go unit tests
.PHONY: test
test: mlmd # a MLMD server running in background is required by some tests
	go test ./...

.PHONY: test-update
test-update: mlmd
	# Updating compiled argo YAML golden files...
	go test ./compiler/argocompiler --args --update

# make test-watch watches file system changes and rerun go unit tests automatically
# install inotifywait by:
# sudo apt-get install inotify-tools
.PHONY: test-watch
test-watch: mlmd
	while inotifywait --exclude .swp -e modify -r .; do \
		go test ./...; \
	done

###### KFP v2 targets ######

# make dev runs developing actions from end to end for KFP v2.
# Build images, push them to dev image registry, build backend compiler, compile pipelines and run them.
.PHONY: dev
dev: pipeline/v2/hello_world \
	pipeline/v2/producer_consumer_param \
	pipeline/test/lightweight_python_functions_v2_pipeline \
	pipeline/test/lightweight_python_functions_v2_with_outputs
# pipeline/core/condition/condition_v2:
# pipeline/core/loop_output/loop_output_v2:
# pipeline/core/loop_static/loop_static_v2:
# pipeline/core/loop_parameter/loop_parameter_v2:
# pipeline/core/resource_spec/resource_spec_v2:
# TODO(v2): migrate v1 samples to v2.
# pipeline/test/two_step \
# pipeline/test/lightweight_python_functions_v2_pipeline
# pipeline/v2/producer_consumer_param \

.PHONY: build image-dev
build: build/compiler build/driver
image-dev: image-launcher-v2-dev image-driver-dev

# Install v2 backend compiler to ~/bin.
# Please add ~/bin to your $PATH first.
.PHONY: install-compiler
install-compiler: build/compiler
	cp build/compiler ~/bin/kfp-v2-compiler

# Write out all possible targets for build/% image-%-dev rules
# for shell autocomplete.

# launcher v2
build/launcher-v2:
image-launcher-v2-dev:

# compiler
build/compiler:

# driver
build/driver:
image-driver-dev:

# tests pipelines
pipeline/v2/hello_world:
pipeline/v2/producer_consumer_param:
pipeline/v2/pipeline_with_importer:
pipeline/test/two_step:
pipeline/test/lightweight_python_functions_v2_pipeline:
pipeline/test/lightweight_python_functions_v2_with_outputs:
pipeline/core/loop_output/loop_output_v2:
pipeline/core/loop_static/loop_static_v2:
pipeline/core/loop_parameter/loop_parameter_v2:
pipeline/core/resource_spec/resource_spec_v2:
pipeline/core/condition/condition_v2:
pipeline/core/condition/condition_v2_simple:

# Run a test pipeline using v2 CLI compiler (v2 engine mode).
.PHONY: pipeline/%
pipeline/%: build/compiler image-dev
	tmp="$$(mktemp -d)" \
	&& mkdir -p "$$(dirname $${tmp}/$*)" \
	&& echo "SDK Compiling to $${tmp}/$*.json" && dsl-compile-v2 --py $(REPO_ROOT)/samples/$*.py --out "$${tmp}/$*.json" \
	&& echo "Backend Compiler compiling to $${tmp}/$*.yaml" && build/compiler --spec "$${tmp}/$*.json" --driver "$(DEV_IMAGE_PREFIX)driver:latest" --launcher "$(DEV_IMAGE_PREFIX)launcher-v2:latest" > "$${tmp}/$*.yaml" \
	&& echo "Linting..." && if which argo >/dev/null; then argo lint "$${tmp}/$*.yaml"; else echo "argo CLI not found, skip linting"; fi \
	&& echo "Running the pipeline..." && kfp run submit -f "$${tmp}/$*.yaml" -e default -r "$*_$${RANDOM}"

###### Common target implementation details ######

# always rebuild, it's hard to set up exact dependencies
# For build/launcher-v2, $@ will be build/launcher-v2, $* will be launcher-v2.
build/%: FORCE
	$(GO_FLAGS) go build $(GO_BUILD_FLAGS) -o $@ github.com/kubeflow/pipelines/v2/cmd/$*
image-%-dev: FORCE
	@KO_DOCKER_REPO=$(DEV_IMAGE_PREFIX)$* ko publish ./cmd/$* --bare
install-ko: FORCE
	go install github.com/google/ko@$(KO_VERSION)

# Run a temporary MLMD server, useful for development / testing.
# This needs to be run in background when running go unit tests.
.PHONY: mlmd
mlmd:
	@if docker ps | grep mlmd; then echo "MLMD container is already running."; else \
	docker run --name mlmd \
		--detach \
		--rm \
		-p 8080:8080 \
		--env MLMD_CONFIG=connection_config{sqlite{filename_uri:\"file:///mlmd.db\"}} \
		--entrypoint /bin/bash \
		"gcr.io/tfx-oss-public/ml_metadata_store_server:${MLMD_VERSION}" \
		-exc 'echo "$${MLMD_CONFIG}" > config && /bin/metadata_store_server -metadata_store_server_config_file config' \
		&& echo "MLMD container starts to run."; \
		fi

# Add the FORCE as a dependency if a target should be run each time.
FORCE: ;
