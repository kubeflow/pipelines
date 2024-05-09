# Copyright 2018, 2021 The Kubeflow Authors
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

REMOTE=gcr.io/ml-pipeline-test/release@sha256:ed1a4dbe536e7e161ad0d846b5681aacc0e0e7f285985cb1808c5c8987bcfeb0

CONTAINER_ENGINE ?= docker

.PHONY: release
release:
	# Usage: TAG=<TAG> BRANCH=<BRANCH> make release
	# Example, release in release branch:
	#   TAG=1.4.0 BRANCH=release-1.4 make release
	# Example, release in master branch:
	#   TAG=2.0.0-rc.1 BRANCH=master make release
	./release.sh $(TAG) $(BRANCH)

.PHONY: release-in-place
release-in-place:
	# Note, please update ../../VERSION file using the following command first:
	# echo -n 1.6.0 > ../../VERSION
	# This script updates the repo according to the VERSION file.
	./bump-version-docker.sh

# Build a release image locally using docker.
.PHONY: build
build:
	${CONTAINER_ENGINE} build -t $(REMOTE) - < Dockerfile.release

# Push locally built release image to remote container registry,
# so that others can use the new image next time.
.PHONY: push
push: build
	# Only some maintainers have access to push,
	# contact @chensun @zijianjoy if you have any needs.
	${CONTAINER_ENGINE} push $(REMOTE)

# Run the docker image interactively in shell as current user.
.PHONY: dev
dev:
	${CONTAINER_ENGINE} run -it -u $$(id -u):$$(id -g) $(REMOTE) /bin/bash
