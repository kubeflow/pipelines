
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

# Component YAML Tests
.PHONY: test-component-yaml-install-deps
test-component-yaml-install-deps:
	python3 -m pip install pytest
	python3 -m pip install pytest-asyncio-cooperative==0.37.0

.PHONY: test-component-yaml-run
test-component-yaml-run:
	./test/presubmit-component-yaml.sh
