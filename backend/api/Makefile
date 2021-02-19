# Makefile to generate KFP api clients from proto.

IMAGE_TAG=kfp-api-generator
# Contact one of Bobgy, capri-xiyue or zijianjoy if this remote image needs an update.
REMOTE_IMAGE=gcr.io/ml-pipeline-test/api-generator
PREBUILT_REMOTE_IMAGE=gcr.io/ml-pipeline-test/api-generator@sha256:2bca5a3e4c1a6c8f4677ef8433ec373894599e35febdc84c4563c2c9bb3f8de7

# Generate clients using a pre-built api-generator image.
.PHONY: generate
generate: hack/generator.sh *.proto
	docker run --interactive --rm \
		--user $$(id -u):$$(id -g) \
		--mount type=bind,source="$$(pwd)/../..",target=/go/src/github.com/kubeflow/pipelines \
		$(PREBUILT_REMOTE_IMAGE) /go/src/github.com/kubeflow/pipelines/backend/api/hack/generator.sh

# Generate clients starting by building api-generator image locally.
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
