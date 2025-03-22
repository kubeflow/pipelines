SHELL := /bin/bash

.PHONY: setup-python
setup-python:
	python3 -m venv .venv && \
	source .venv/bin/activate

.PHONY: test-backend-visualization-test
test-backend-visualization-test:
	./test/presubmit-backend-visualization.sh

# Check diff for generated files
.PHONY: check-diff
check-diff:
	/bin/bash -c 'if [[ -n "$$(git status --porcelain)" ]]; then \
		echo "ERROR: Generated files are out of date"; \
		echo "Please regenerate using make clean all for api and kubernetes_platform"; \
		echo "Changes found in the following files:"; \
		git status; \
		echo "Diff of changes:"; \
		git diff; \
		exit 1; \
	fi'
