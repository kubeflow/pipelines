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

# create an .env file to override dev vars locally
-include .env
LAUNCHER_IMAGE=gcr.io/ml-pipeline/kfp-launcher
LAUNCHER_IMAGE_DEV?=gcr.io/ml-pipeline-test/dev/kfp-launcher

build-launcher:
	@docker build -t ${LAUNCHER_IMAGE} -f launcher_container/Dockerfile .

push-launcher: build-launcher
	@docker push ${LAUNCHER_IMAGE}

all: push-launcher

# The following are dev rules:

dev-build-launcher: build/launcher launcher_container/Dockerfile.dev launcher_container/mount_launcher.sh
	@docker build -t ${LAUNCHER_IMAGE_DEV} -f launcher_container/Dockerfile.dev ./

dev-push-launcher: dev-build-launcher
	@docker push ${LAUNCHER_IMAGE_DEV}

# always rebuild, it's hard to set up exact dependencies
.PHONY: build/launcher
build/launcher:
	GO111MODULE=on \
	CGO_ENABLED=0 \
	GOOS=linux \
	GOARCH=amd64 \
	go build -tags netgo -ldflags '-extldflags "-static"' -o build/launcher github.com/kubeflow/pipelines/v2/cmd/launch

.PHONY: start-mlmd
start-mlmd:
# Run a temporary MLMD server, useful for development / testing.
	docker run -p 8080:8080 \
		--env MLMD_CONFIG=connection_config{sqlite{filename_uri:\"file:///mlmd.db\"}} \
		--entrypoint /bin/bash \
		gcr.io/tfx-oss-public/ml_metadata_store_server:0.30.0 \
		-exc 'echo "$${MLMD_CONFIG}" > config && /bin/metadata_store_server -metadata_store_server_config_file config'
