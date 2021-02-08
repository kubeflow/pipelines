REPO_ROOT=../../../..
GO_COMMON_ARGS=--logtostderr
DRIVER_EXAMPLE_PATH=./driver/example
PUBLISHER_EXAMPLE_PATH=./publisher/example
DATA_PATH=./tmp
DRIVER_CODE=./driver/*.go
PUBLISHER_CODE=./publisher/*.go
ENTRYPOINT_CODE=./entrypoint/*.go

.PHONY: run-producer-consumer
run-producer-consumer: $(DRIVER_CODE) $(ENTRYPOINT_CODE) $(PUBLISHER_CODE) $(DRIVER_EXAMPLE_PATH)/task_spec_producer.json $(DRIVER_EXAMPLE_PATH)/task_spec_dag.json
	# Make sure you have a separate running terminal that runs
	# `make proxy`.

	# Clean up intermediate data
	rm -rf $(DATA_PATH)
	@echo
	# ====== Run DAG driver ==========
	go run $(DRIVER_CODE) $(GO_COMMON_ARGS) \
		--task_spec="$$(cat $(DRIVER_EXAMPLE_PATH)/task_spec_dag.json)" \
		--execution_name="kfp-$$RANDOM" \
		--driver_type=DAG \
		--output_path_execution_id="$(DATA_PATH)/dag/execution_id" \
		--output_path_context_name="$(DATA_PATH)/dag/context_name"
	@if ls $(DATA_PATH)/dag/execution_id $(DATA_PATH)/dag/context_name; then \
		echo "DAG driver outputs are found."; \
	else \
		echo "Some of DAG driver outputs are missing." \
		exit 1; \
	fi
	@echo
	# ======== Run producer task driver ========
	go run $(DRIVER_CODE) $(GO_COMMON_ARGS) \
		--task_spec="$$(cat $(DRIVER_EXAMPLE_PATH)/task_spec_producer.json)" \
		--executor_spec="$$(cat $(DRIVER_EXAMPLE_PATH)/executor_spec_producer.json)" \
		--execution_name="kfp-$$RANDOM" \
		--driver_type=EXECUTOR \
		--parent_context_name="$$(cat $(DATA_PATH)/dag/context_name)" \
		--output_path_execution_id="$(DATA_PATH)/producer/execution_id" \
		--output_path_pod_spec_patch="$(DATA_PATH)/producer/pod_spec_patch.json"
	cat $(DATA_PATH)/producer/pod_spec_patch.json
	@echo
	@echo
	# ========= Run producer task entrypoint =======
	mkdir -p $(DATA_PATH)/producer/parameters
	go run $(ENTRYPOINT_CODE) $(GO_COMMON_ARGS) \
		--component_outputs_spec="$$(cat $(PUBLISHER_EXAMPLE_PATH)/outputs_spec.json)" \
		--execution_id=$$(cat $(DATA_PATH)/producer/execution_id) \
		--publisher_type=EXECUTOR \
		--input_path_parameters=$(DATA_PATH)/producer/parameters \
		-- "bash" "-c" "echo output value > $(DATA_PATH)/producer/parameters/output_value"
	@echo
	# ========= Run consumer task driver ==========
	go run $(DRIVER_CODE) $(GO_COMMON_ARGS) \
		--task_spec="$$(cat $(DRIVER_EXAMPLE_PATH)/task_spec_consumer.json)" \
		--executor_spec="$$(cat $(DRIVER_EXAMPLE_PATH)/executor_spec_consumer.json)" \
		--execution_name="kfp-$$RANDOM" \
		--driver_type=EXECUTOR \
		--parent_context_name="$$(cat $(DATA_PATH)/dag/context_name)" \
		--output_path_execution_id="$(DATA_PATH)/consumer/execution_id" \
		--output_path_pod_spec_patch="$(DATA_PATH)/consumer/pod_spec_patch.json"
	cat $(DATA_PATH)/consumer/pod_spec_patch.json
	@echo

.PHONY: image-push
image-push:
	cd driver && $(MAKE) image-push
	cd publisher && $(MAKE) image-push

.PHONY: proxy
proxy:
	kubectl port-forward -n kubeflow svc/metadata-grpc-service 8080:8080
