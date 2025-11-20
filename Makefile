
# Regenerate all generated files and golden test files
.PHONY: regenerate-all
regenerate-all:
	@echo "==> Regenerating K8s Native API CRDs..."
	cd backend/src/crd/kubernetes && $(MAKE) generate manifests
	@echo "==> Regenerating backend proto code (v2beta1)..."
	cd backend/api && API_VERSION=v2beta1 $(MAKE) generate
	@echo "==> Regenerating backend proto code (v1beta1)..."
	cd backend/api && API_VERSION=v1beta1 $(MAKE) generate
	@echo "==> Regenerating kfp-server-api-package (v2beta1)..."
	cd backend/api && API_VERSION=v2beta1 $(MAKE) generate-kfp-server-api-package
	@echo "==> Regenerating kfp-server-api-package (v1beta1)..."
	cd backend/api && API_VERSION=v1beta1 $(MAKE) generate-kfp-server-api-package
	@echo "==> Updating proto test golden files..."
	cd backend/test/proto_tests && UPDATE_EXPECTED=true go test .
	@echo "==> Updating compiler test golden files..."
	cd backend/test/compiler && go test . -updateCompiledFiles
	@echo "==> All files regenerated successfully!"

# Check diff for generated files
.PHONY: check-diff
check-diff:
	/bin/bash -c 'if [[ -n "$$(git status --porcelain)" ]]; then \
		echo "ERROR: Generated files are out of date"; \
		echo "Please regenerate using: make regenerate-all"; \
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
