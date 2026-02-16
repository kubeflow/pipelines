
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

# GCPC modules tests
.PHONY: install-python-build-tools
install-python-build-tools:
	pip3 install setuptools build wheel==0.42.0

.PHONY: install-kfp-pipeline-spec
install-kfp-pipeline-spec:
	python3 -m pip install -I api/v2alpha1/python

.PHONY: install-kfp-server-api
install-kfp-server-api:
	cd backend/api/v2beta1/python_http_client && \
	rm -rf dist/ && \
	python -m build . && \
	pip install dist/*.whl

.PHONY: install-sdk
install-sdk:
	pip install sdk/python

.PHONY: install-gcpc-components
install-gcpc-components:
	pip install components/google-cloud

.PHONY: install-pytest
install-pytest:
	pip install "$$(grep 'pytest==' sdk/python/requirements-dev.txt)"

.PHONY: test-gcpc-modules
test-gcpc-modules:
	pytest ./test/gcpc-tests/run_all_gcpc_modules.py
