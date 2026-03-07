
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

# Tools
BIN_DIR ?= $(CURDIR)/bin

.PHONY: ginkgo
ginkgo:
	mkdir -p $(BIN_DIR)
	GOBIN=$(BIN_DIR) go install github.com/onsi/ginkgo/v2/ginkgo@latest
	@echo "Ginkgo installed to $(BIN_DIR)/ginkgo"

# GCPC Module Tests
.PHONY: test-gcpc-modules
test-gcpc-modules:
	pytest ./test/gcpc-tests/run_all_gcpc_modules.py

# Install GCPC test dependencies (kfp SDK, google-cloud components, pytest)
.PHONY: install-gcpc-test-deps
install-gcpc-test-deps:
	pip install sdk/python
	pip install components/google-cloud
	pip install "$$(grep 'pytest==' sdk/python/requirements-dev.txt)"
