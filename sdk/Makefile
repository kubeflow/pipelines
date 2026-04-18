# Copyright 2025 The Kubeflow Authors
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
PREBUILT_REMOTE_IMAGE=ghcr.io/kubeflow/kfp-api-generator:master
UV_VERSION=0.10.3

.PHONY: python
python:
	docker run --interactive --rm \
		--user $$(id -u):$$(id -g) \
		-e HOME=/tmp \
		-v "$$(pwd)/..":"/go/src/github.com/kubeflow/pipelines":z \
		$(PREBUILT_REMOTE_IMAGE) \
		sh -c 'cd /go/src/github.com/kubeflow/pipelines && \
			pip install --user --break-system-packages uv==$(UV_VERSION) && \
			~/.local/bin/uv build --package kfp --out-dir sdk/python/dist'

.PHONY: clean-python
clean-python:
	rm -rf python/build
	rm -rf python/dist
	rm -rf python/kfp.egg-info
