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

# Pre-built images for faster development (default: enabled for speed)
# Set USE_PREBUILT_IMAGE=false when:
# - Dockerfile dependencies have been updated
# - CI validation fails due to tool version mismatches
# - Working on API generation toolchain changes
USE_PREBUILT_IMAGE ?= true
# Validate USE_PREBUILT_IMAGE value
ifneq ($(USE_PREBUILT_IMAGE),true)
ifneq ($(USE_PREBUILT_IMAGE),false)
$(error USE_PREBUILT_IMAGE must be 'true' or 'false', got '$(USE_PREBUILT_IMAGE)')
endif
endif
PREBUILT_REMOTE_IMAGE=ghcr.io/kubeflow/kfp-api-generator:master
IMAGE_TAG=kfp-api-generator

# Build Python package using either pre-built image (fast) or source build (accurate).
# Use USE_PREBUILT_IMAGE=false to build from source for guaranteed accuracy.
.PHONY: python
python:
ifeq ($(USE_PREBUILT_IMAGE),true)
	docker run --interactive --rm \
		--user $$(id -u):$$(id -g) \
		-e HOME=/tmp \
		-v "$$(pwd)/..":"/go/src/github.com/kubeflow/pipelines":z \
		$(PREBUILT_REMOTE_IMAGE) \
		sh -c 'cd /go/src/github.com/kubeflow/pipelines/sdk/python && \
			python3 setup.py sdist && pip wheel --no-deps dist/*.tar.gz -w dist'
else
	$(MAKE) python-from-source
endif

# Build Python package by building api-generator image from source.
.PHONY: python-from-source
python-from-source: ensure-api-generator-image
	docker run --interactive --rm \
		--user $$(id -u):$$(id -g) \
		-e HOME=/tmp \
		-v "$$(pwd)/..":"/go/src/github.com/kubeflow/pipelines":z \
		$(IMAGE_TAG) \
		sh -c 'cd /go/src/github.com/kubeflow/pipelines/sdk/python && \
			python3 setup.py sdist && pip wheel --no-deps dist/*.tar.gz -w dist'

.PHONY: clean-python
clean-python:
	rm -rf python/build
	rm -rf python/dist
	rm -rf python/kfp.egg-info

# Build the API generator image via the canonical target in backend/api
.PHONY: ensure-api-generator-image
ensure-api-generator-image:
	$(MAKE) -C ../backend/api image
