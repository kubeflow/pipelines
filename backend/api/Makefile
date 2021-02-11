IMAGE_TAG=kfp-api-generator

.PHONY: generate_api
generate_api: .image-built
	# Generate API
	docker run  --interactive --rm \
	    --user $$(id -u):$$(id -g) \
		--mount type=bind,source="$$(pwd)/../..",target=/go/src/github.com/kubeflow/pipelines \
		$(IMAGE_TAG) /go/src/github.com/kubeflow/pipelines/backend/api/hack/generator.sh

.PHONY: image
image: .image-built

.image-built: Dockerfile
	docker build ../.. -t $(IMAGE_TAG) -f Dockerfile
	touch .image-built
