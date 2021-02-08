REPO_ROOT=../../../..
GO_COMMON_ARGS=--logtostderr

# Example .env file:
# ```
# IMAGE_DEV=gcr.io/gongyuan-pipeline-test/kfp-publisher
# ```
# It should be an image repository where you can push to via docker push.
include .env

.PHONY: run-producer
run-producer: *.go ./example/task_spec_dag.json ./example/task_spec_producer.json ./example/executor_spec_producer.json
	# Make sure you have a separate running terminal that runs
	# `make proxy`.

	# Clean up intermediate data
	rm -rf ./data
	# Run DAG driver
	go run *.go $(GO_COMMON_ARGS) \
		--task_spec="$$(cat ./example/task_spec_dag.json)" \
		--execution_name="kfp-$$RANDOM" \
		--driver_type=DAG \
		--output_path_execution_id="./data/execution_id" \
		--output_path_context_name="./data/context_name"
	# Run task driver
	@if ls ./example/task_spec_producer.json ./data/execution_id ./data/context_name; then \
		echo "inputs are found"; \
	else \
	    exit 1; \
	fi
	go run *.go $(GO_COMMON_ARGS) \
		--task_spec="$$(cat ./example/task_spec_producer.json)" \
		--executor_spec="$$(cat ./example/executor_spec_producer.json)" \
		--execution_name="kfp-$$RANDOM" \
		--driver_type=EXECUTOR \
		--parent_context_name="$$(cat ./data/context_name)" \
		--output_path_execution_id="./data/execution_id" \
		--output_path_pod_spec_patch="./data/pod_spec_patch.json"
	cat ./data/pod_spec_patch.json

.PHONY: run
run: *.go example/task_spec_dag.json
	# Make sure you have a separate running terminal that runs
	# `make proxy`.

	# Clean up intermediate data
	rm -rf ./data
	# Run DAG driver
	go run *.go $(GO_COMMON_ARGS) \
		--task_spec="$$(cat ./example/task_spec_dag.json)" \
		--execution_name="kfp-$$RANDOM" \
		--driver_type=DAG \
		--output_path_execution_id="./data/execution_id" \
		--output_path_context_name="./data/context_name"
	# Run task driver
	@if ls ./example/task_spec_hw.json ./data/execution_id ./data/context_name; then \
		echo "inputs are found"; \
	else \
	    exit 1; \
	fi
	go run *.go $(GO_COMMON_ARGS) \
		--task_spec="$$(cat ./example/task_spec_hw.json)" \
		--executor_spec="$$(cat ./example/executor_spec_hw.json)" \
		--execution_name="kfp-$$RANDOM" \
		--driver_type=EXECUTOR \
		--parent_context_name="$$(cat ./data/context_name)" \
		--output_path_execution_id="./data/execution_id" \
		--output_path_pod_spec_patch="./data/pod_spec_patch.json"
	cat ./data/pod_spec_patch.json

.PHONY: run-docker
run-docker: .build
	docker run kfp/driver $(GO_COMMON_ARGS) \
		--task_spec="$$(cat ./example/task_spec_dag.json)" \
		--execution_name="kfp-$$RANDOM" \
		--driver_type=DAG \
		--output_path_execution_id="./data/execution_id" \
		--output_path_context_name="./data/context_name" \
		--mlmd_url="host.docker.internal:8080"

.PHONY: proxy
proxy:
	kubectl port-forward -n kubeflow svc/metadata-grpc-service 8080:8080

.PHONY: example
example: example/task_spec_hw.json

.PHONY: image
image: .build

.PHONY: image-push
image-push: image
	docker push $(IMAGE_DEV)

.build: Dockerfile.dev main .env
	docker build . -f Dockerfile.dev -t kfp/driver -t $(IMAGE_DEV)
	touch .build

main: *.go $(REPO_ROOT)/go.mod
	env GOOS=linux CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' -o main *.go

example/task_spec_dag.json: example/*.go
	go run example/*.go

example/task_spec_hw.json: example/*.go
	go run example/*.go
