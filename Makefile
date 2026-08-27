
# Check diff for generated files. Changes confined to the generator version
# comment that protoc-gen-go and protoc-gen-go-grpc stamp into their output are
# reported and allowed, because those versions come from the Go module manifests
# and so change on any module bump without the generated code differing.
.PHONY: check-diff
check-diff:
	python3 .github/resources/scripts/check_generated_diff.py

.PHONY: check-go-version
check-go-version:
	python3 .github/resources/scripts/go_version_consistency_test.py

.PHONY: update-go-version
update-go-version:
	@if [ -z "$(GO_VERSION)" ]; then \
		echo "GO_VERSION is required; run make update-go-version GO_VERSION=1.X.Y" >&2; \
		exit 2; \
	fi
	python3 .github/resources/scripts/update_go_version.py --version "$(GO_VERSION)"

# Tools
BIN_DIR ?= $(CURDIR)/bin

.PHONY: ginkgo
ginkgo:
	mkdir -p $(BIN_DIR)
	GOBIN=$(BIN_DIR) go install github.com/onsi/ginkgo/v2/ginkgo@latest
	@echo "Ginkgo installed to $(BIN_DIR)/ginkgo"

# Backend visualization tests
.PHONY: install-backend-visualization-deps
install-backend-visualization-deps:
	cd backend/src/apiserver/visualization && \
	python3 -m pip install --upgrade pip && \
	python3 -m pip install -r requirements.txt -r requirements-test.txt

.PHONY: test-backend-visualization
test-backend-visualization:
	cd backend/src/apiserver/visualization && \
	python3 test_exporter.py && \
	python3 test_server.py
