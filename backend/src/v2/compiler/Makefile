REPO_ROOT=../../../..
# PIPELINE_SPEC_PATH=$(REPO_ROOT)/samples/v2/hello_world/hw_pipeline_job.json
PIPELINE_SPEC_PATH=$(REPO_ROOT)/samples/v2/producer_consumer_param/two_step_pipeline_job.json
SAMPLE_PIPELINE_SPEC=../driver/example/task_spec_dag.json

run: compile $(SAMPLE_PIPELINE_SPEC)
	# Run compiled argo workflow spec in workflow.yaml in an existing KFP cluster
	# using argo CLI to submit.
	#
	# Alternatively, you can upload this workflow.yaml in KFP UI and run it.
	argo submit --watch workflow.yaml \
		--serviceaccount pipeline-runner \
		--namespace kubeflow \
		--parameter task-spec="$$(cat $(SAMPLE_PIPELINE_SPEC))"

# TODO(Bobgy): How can we write ./**/*.go instead?
.PHONY: compile
compile: ./*.go ./templates/*.go
	set -o pipefail \
		&& ( \
			go run ./*.go --pipeline_spec_path=$(PIPELINE_SPEC_PATH) | tee workflow.yaml \
			|| (rm -f workflow.yaml && exit 1) \
		)
