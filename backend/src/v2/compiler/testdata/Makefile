REPO_ROOT=../../../../..
V2_SAMPLES=$(REPO_ROOT)/samples/v2
CORE_SAMPLES=$(REPO_ROOT)/samples/core

# Bobgy: I decided to commit compiled samples into the repo, because they are
# used by compiler unit tests. Even if there are updates to v2 DSL compiler
# or samples, we don't necessarily want to update the compiled samples here,
# because:
# 1. outdated samples are still good data to confirm we keep backward compatibility
# 2. if we compile python DSL to JSON pipeline spec files during unit test, unit
# test env is harder to configure with both python and golang dev env.
# Instead, let's update testdata and commit them to repo again when needed.

# Update compiled JSON pipeline spec files by compiling v2 samples using dsl-compile-v2 CLI.
# Add more samples here when needed
.PHONY: update
update:
	dsl-compile-v2 --py "$(V2_SAMPLES)/hello_world.py" --output "hello_world.json"
	dsl-compile-v2 --py "$(V2_SAMPLES)/producer_consumer_param.py" --output "producer_consumer_param.json"
	dsl-compile-v2 --py "$(CORE_SAMPLES)/exit_handler/exit_handler.py" --output "exit_handler.json"
	dsl-compile-v2 --py "nested_pipeline_all_level_retry.py" --output "nested_pipeline_all_level_retry.json"
	dsl-compile-v2 --py "nested_pipeline_pipeline_retry.py" --output "nested_pipeline_pipeline_retry.json"
	dsl-compile-v2 --py "nested_pipeline_sub_component_retry.py" --output "nested_pipeline_sub_component_retry.json"
	# currently commented, because v2 compiler generates duplicate component definitions
	# the commited component_used_twice.json file is hand edited.
	# dsl-compile-v2 --py "component_used_twice.py" --output "component_used_twice.json"
