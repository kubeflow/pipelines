# Copyright 2021 Google LLC
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

LAUNCHER_IMAGE=gcr.io/ml-pipeline/kfp-launcher

build-launcher:
	@docker build -t ${LAUNCHER_IMAGE} -f launcher_container/Dockerfile .

push-launcher: build-launcher
	@docker push ${LAUNCHER_IMAGE}

all: push-launcher

.PHONY: start-mlmd
start-mlmd:
# Run a temporary MLMD server, useful for development / testing.
	docker run -p 8080:8080 \
		--env MLMD_CONFIG=connection_config{sqlite{filename_uri:\"file:///mlmd.db\"}} \
		--entrypoint /bin/bash \
		gcr.io/tfx-oss-public/ml_metadata_store_server:0.25.1 \
		-exc 'echo "$${MLMD_CONFIG}" > config && /bin/metadata_store_server -metadata_store_server_config_file config'
