REPO_ROOT=../../../..
GO_COMMON_ARGS=--logtostderr
IMAGE=kfp-entrypoint
EXECUTION_ID=160
PUBLISHER_EXAMPLES=../publisher/example

# Example .env file:
# ```
# IMAGE_DEV=gcr.io/gongyuan-pipeline-test/kfp-publisher
# ```
# It should be an image repository where you can push to via docker push.
include .env

.PHONY: run
run: *.go
	# Make sure you have a separate running terminal that runs
	# `make proxy`.

	# Clean up intermediate data
	rm -rf ./data
	# Run executor publisher
	go run *.go $(GO_COMMON_ARGS) \
		--component_outputs_spec="$$(cat $(PUBLISHER_EXAMPLES)/outputs_spec.json)" \
		--execution_id=$(EXECUTION_ID) \
		--publisher_type=EXECUTOR \
		--input_path_parameters=$(PUBLISHER_EXAMPLES)/parameters \
		-- echo -n Hello, World!

.PHONY: run-docker
run-docker: .build
	docker run $(IMAGE) /bin/kfp-entrypoint $(GO_COMMON_ARGS) \
		--component_outputs_spec="$$(cat $(PUBLISHER_EXAMPLES)/outputs_spec.json)" \
		--execution_id=$(EXECUTION_ID) \
		--publisher_type=EXECUTOR \
		--input_path_parameters=$(PUBLISHER_EXAMPLES)/parameters \
		-- echo -n Hello, World!

.PHONY: proxy
proxy:
	kubectl port-forward -n kubeflow svc/metadata-grpc-service 8080:8080

.PHONY: image
image: .build

.PHONY: image-push
image-push: image
	docker push $(IMAGE_DEV)

.build: Dockerfile.dev main .env
	docker build . -f Dockerfile.dev -t $(IMAGE) -t $(IMAGE_DEV)
	touch .build

main: *.go $(REPO_ROOT)/go.mod
	env GOOS=linux CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' -o main *.go
