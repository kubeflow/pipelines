# Copyright 2018 Google LLC
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

REMOTE=gcr.io/ml-pipeline-test/release:latest

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

.PHONY: build
build:
	docker build -t $(REMOTE) - < Dockerfile.release

.PHONY: push
push: build
	# Only some maintainers have access to push,
	# contact @Bobgy @capri-xiyue @zijianjoy if you have any needs.
	docker push $(REMOTE)

.PHONY: dev
dev:
	# Run the docker image interactively in shell as current user.
	docker run -it -u $$(id -u):$$(id -g) $(REMOTE) /bin/bash
