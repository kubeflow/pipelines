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

IMG = gcr.io/ml-pipeline-test/image-builder

# List any changed  files. We only include files in the notebooks directory.
# because that is the code in the docker image.
# In particular we exclude changes to the ksonnet configs.
DIR := ${CURDIR}
CHANGED_FILES := $(shell git diff-files --relative=$(DIR))

GIT_VERSION := $(shell git describe --always)
ifneq ($(strip $(CHANGED_FILES)),)
GIT_VERSION := $(GIT_VERSION)-dirty-$(shell git diff | shasum -a256 | cut -c -6)
endif

TAG := $(shell date +v%Y%m%d)-$(GIT_VERSION)
all: build

# To build without the cache set the environment variable
# export DOCKER_BUILD_OPTS=--no-cache
build:
	docker build ${DOCKER_BUILD_OPTS} -t $(IMG):$(TAG) . \
           --label=git-verions=$(GIT_VERSION)
	docker tag $(IMG):$(TAG) $(IMG):latest
	echo Built $(IMG):$(TAG)
	echo Built $(IMG):latest

# Build but don't attach the latest tag. This allows manual testing/inspection of the image
# first.
push: build
	gcloud auth configure-docker
	docker push $(IMG):$(TAG)
	echo Pushed $(IMG):$(TAG)

push-latest: push
	gcloud container images add-tag --quiet $(IMG):$(TAG) $(IMG):latest --verbosity=info
	echo Updated $(IMG):latest
