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

IMG = gcr.io/ml-pipeline-test/selenium-standalone-chrome-gcloud-nodejs

# Unique tag including date, commit SHA and a random number
TAG := $(shell date +v%Y%m%d)-$(shell git describe --tags --always --dirty)-$(shell git diff | shasum -a256 | cut -c -6)

all: build

# To build without the cache set the environment variable
# export DOCKER_BUILD_OPTS=--no-cache
build:
	docker build ${DOCKER_BUILD_OPTS} -t $(IMG):$(TAG) .
	@echo Built $(IMG):$(TAG)

# Build but don't attach the latest tag. This allows manual testing/inspection of the image first.
push: build
	gcloud auth configure-docker
	docker push $(IMG):$(TAG)
	@echo Pushed $(IMG) with  :$(TAG) tags
	# Tagging into a new project can be very slow but it appears to work.
	gcloud container images add-tag --quiet $(IMG):$(TAG) $(IMG):$(TAG) --verbosity=info
	echo created $(IMG):$(TAG)

push-latest: push
	gcloud container images add-tag --quiet $(IMG):$(TAG) $(IMG):latest --verbosity=info
	echo created $(IMG):latest
