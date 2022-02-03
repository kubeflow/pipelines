REPO_ROOT=../../..
V2_SAMPLES=$(REPO_ROOT)/samples/v2

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
	dsl-compile-v2 --py "$(V2_SAMPLES)/hello_world.py" --out "hello_world.json"
	dsl-compile-v2 --py "$(V2_SAMPLES)/producer_consumer_param.py" --out "producer_consumer_param.json"
	# currently commented, because v2 compiler generates duplicate component definitions
	# the commited component_used_twice.json file is hand edited.
	# dsl-compile-v2 --py "component_used_twice.py" --out "component_used_twice.json"
