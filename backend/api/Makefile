# Copyright 2021 The Kubeflow Authors
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

# Makefile to generate KFP api clients from proto.

IMAGE_TAG=kfp-api-generator
# Contact one of Bobgy, or zijianjoy if this remote image needs an update.
REMOTE_IMAGE=gcr.io/ml-pipeline-test/api-generator
PREBUILT_REMOTE_IMAGE=gcr.io/ml-pipeline-test/api-generator@sha256:10a09669794180b91c8b0fffb803db81eb458372e6ab669051f40d4979cdf43c

# Generate clients using a pre-built api-generator image.
.PHONY: generate
generate: hack/generator.sh *.proto
	docker run --interactive --rm \
		--user $$(id -u):$$(id -g) \
		--mount type=bind,source="$$(pwd)/../..",target=/go/src/github.com/kubeflow/pipelines \
		$(PREBUILT_REMOTE_IMAGE) /go/src/github.com/kubeflow/pipelines/backend/api/hack/generator.sh

# Generate clients starting by building api-generator image locally.
# Note, this should only be used for local development purposes. Once any change is made to the Dockerfile,
# we should push the new image remotely to ensure everyone is using the same tools.
.PHONY: generate-from-scratch
generate-from-scratch: .image-built hack/generator.sh *.proto
	docker run  --interactive --rm \
	    --user $$(id -u):$$(id -g) \
		--mount type=bind,source="$$(pwd)/../..",target=/go/src/github.com/kubeflow/pipelines \
		$(IMAGE_TAG) /go/src/github.com/kubeflow/pipelines/backend/api/hack/generator.sh

# Build a local api-generator image.
.PHONY: image
image: .image-built

# Push api-generator image remotely.
.PHONY: push
push: image
	docker tag $(IMAGE_TAG) $(REMOTE_IMAGE)
	docker push $(REMOTE_IMAGE)

# .image-built is a local token file to help Make determine the latest successful build.
.image-built: Dockerfile
	docker build ../.. -t $(IMAGE_TAG) -f Dockerfile
	touch .image-built
