IMAGE_TAG=kfp-api-generator

.PHONY: generate_api
generate_api: .image-built
	# Generate API
	docker run  --interactive --rm \
		--mount type=bind,source="$$(pwd)/../..",target=/go/src/github.com/kubeflow/pipelines \
		$(IMAGE_TAG) /go/src/github.com/kubeflow/pipelines/backend/api/hack/generator.sh

	# Change owner to user for generate files, explanation of command: https://askubuntu.com/questions/829537/how-do-i-change-owner-to-current-user-on-folder-and-containing-folders-inside-my
	sudo find . -user root -exec sudo chown $$USER: {} +

.PHONY: image
image: .image-built

.image-built: Dockerfile
	docker build ../.. -t $(IMAGE_TAG) -f Dockerfile
	touch .image-built
