REPO_ROOT=../../../..
GO_COMMON_ARGS=--logtostderr
EXECUTION_ID=160
IMAGE=kfp-publisher

# Example .env file:
# ```
# IMAGE_DEV=gcr.io/gongyuan-pipeline-test/kfp-publisher
# ```
# It should be an image repository where you can push to via docker push.
include .env

.PHONY: run
run: *.go example/outputs_spec.json
	# Make sure you have a separate running terminal that runs
	# `make proxy`.

	# Clean up intermediate data
	rm -rf ./data
	# Run executor publisher
	go run *.go $(GO_COMMON_ARGS) \
		--component_outputs_spec="$$(cat ./example/outputs_spec.json)" \
		--execution_id=$(EXECUTION_ID) \
		--publisher_type=EXECUTOR \
		--input_path_parameters=./example/parameters

.PHONY: run-docker
run-docker: .build
	docker run kfp/publisher $(GO_COMMON_ARGS) \
		--component_outputs_spec="$$(cat ./example/outputs_spec.json)" \
		--execution_id=$(EXECUTION_ID) \
		--publisher_type=EXECUTOR \
		--input_path_parameters=./example/parameters

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
