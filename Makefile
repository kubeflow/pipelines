
# Check diff for generated files. Changes confined to the generator version
# comment that protoc-gen-go and protoc-gen-go-grpc stamp into their output are
# reported and allowed, because those versions come from the Go module manifests
# and so change on any module bump without the generated code differing.
.PHONY: check-diff
check-diff:
	python3 .github/resources/scripts/check_generated_diff.py

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
