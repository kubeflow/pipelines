IMAGE_TAG=kfp-api-generator

.PHONY: generate
generate: .image-built hack/generator.sh *.proto
	docker run  --interactive --rm \
	    --user $$(id -u):$$(id -g) \
		--mount type=bind,source="$$(pwd)/../..",target=/go/src/github.com/kubeflow/pipelines \
		$(IMAGE_TAG) /go/src/github.com/kubeflow/pipelines/backend/api/hack/generator.sh

.PHONY: image
image: .image-built

# .image-built is a local token file to help Make determine the latest successful build.
.image-built: Dockerfile
	docker build ../.. -t $(IMAGE_TAG) -f Dockerfile
	touch .image-built
