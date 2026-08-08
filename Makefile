
SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c

# Tools
BIN_DIR ?= $(CURDIR)/bin
GINKGO_VERSION ?= v2.31.0

# Check diff for generated files
.PHONY: check-diff
check-diff:
	@if [[ -n "$$(git status --porcelain)" ]]; then \
		echo "ERROR: Generated files are out of date"; \
		echo "Please regenerate using 'make clean all' for api and kubernetes_platform"; \
		echo "Changes found in the following files:"; \
		git status; \
		echo "Diff of changes:"; \
		git diff HEAD; \
		exit 1; \
	fi

.PHONY: ginkgo
ginkgo:
	@mkdir -p $(BIN_DIR)
	GOBIN=$(BIN_DIR) go install github.com/onsi/ginkgo/v2/ginkgo@$(GINKGO_VERSION)
	@echo "Ginkgo installed to $(BIN_DIR)/ginkgo"

# Backend visualization tests
.PHONY: install-backend-visualization-deps
install-backend-visualization-deps:
	cd backend/src/apiserver/visualization && \
	python3 -m venv .venv && \
	. .venv/bin/activate && \
	python3 -m pip install --upgrade pip && \
	python3 -m pip install -r requirements.txt -r requirements-test.txt

.PHONY: test-backend-visualization
test-backend-visualization:
	cd backend/src/apiserver/visualization && \
	. .venv/bin/activate && \
	python3 -m unittest test_exporter.py test_server.py