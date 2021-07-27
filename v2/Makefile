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
LAUNCHER_IMAGE?=gcr.io/ml-pipeline/kfp-launcher
DEV_IMAGE_PREFIX?=gcr.io/ml-pipeline-test/dev/kfp-
REPO_ROOT=..
# override this via env to increase verbosity of go-licenses, example:
# GO_LICENSES_FLAGS=-v4 make license-launcher
GO_LICENSES_FLAGS?=
GO_FLAGS=GO111MODULE=on CGO_ENABLED=0 GOOS=linux GOARCH=amd64
GO_BUILD_FLAGS=-tags netgo -ldflags '-extldflags "-static"'

all: push-launcher

image-launcher:
	@docker build -t ${LAUNCHER_IMAGE} -f container/launcher/Dockerfile .
push-launcher: image-launcher
	@docker push ${LAUNCHER_IMAGE}

license-launcher: third_party/licenses/launcher.csv
third_party/licenses/launcher.csv: build/launcher go-licenses.yaml
	go-licenses $(GO_LICENSES_FLAGS) csv build/launcher > third_party/licenses/launcher.csv

.PHONY: start-mlmd
start-mlmd:
# Run a temporary MLMD server, useful for development / testing.
	docker run -p 8080:8080 \
		--env MLMD_CONFIG=connection_config{sqlite{filename_uri:\"file:///mlmd.db\"}} \
		--entrypoint /bin/bash \
		gcr.io/tfx-oss-public/ml_metadata_store_server:1.0.0 \
		-exc 'echo "$${MLMD_CONFIG}" > config && /bin/metadata_store_server -metadata_store_server_config_file config'

.PHONY: build push-dev
build: build/launcher build/compiler build/driver
push-dev: push-launcher-dev push-driver-dev

# always rebuild, it's hard to set up exact dependencies
# For build/launcher, $@ will be build/launcher, $* will be launcher.
build/%: FORCE
	$(GO_FLAGS) go build $(GO_BUILD_FLAGS) -o $@ github.com/kubeflow/pipelines/v2/cmd/$*
image-%-dev: FORCE build/%
	@docker build -t $(DEV_IMAGE_PREFIX)$* -f container/$*/Dockerfile.dev ./
push-%-dev: FORCE image-%-dev
	@docker push ${DEV_IMAGE_PREFIX}$*

.PHONY: install-compiler
install-compiler: build/compiler
	cp build/compiler ~/bin/kfp-compiler

.PHONY: test-%
test-%:
	tmp="$$(mktemp -d)" \
	&& dsl-compile-v2 --py $(REPO_ROOT)/samples/v2/$*.py --out "$${tmp}/$*.json" \
	&& kfp-compiler --spec "$${tmp}/$*.json" > "$${tmp}/$*.yaml" \
	&& kfp run submit -f "$${tmp}/$*.yaml" -e default -r "$*_$${RANDOM}"

.PHONY: dev
dev: push-dev install-compiler test-hello_world test-producer_consumer_param

# Write out all possibilities so that shell autocomplete works
# launcher
build/launcher:
image-launcher-dev:
push-launcher-dev:

# compiler
build/compiler:

# driver
build/driver:
image-driver-dev:
push-driver-dev:

# tests
test-hello_world:
test-producer_consumer_param:

# Add the FORCE as a dependency if a target should be run each time.
FORCE: ;
