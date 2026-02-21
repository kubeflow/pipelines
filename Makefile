
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

# E2E Tests
E2E_TEST_DIR ?= backend/test/end2end
NUM_NODES ?= 1
LABEL_FILTER ?= E2ECritical
NAMESPACE ?= kubeflow
MULTI_USER ?= false
USE_PROXY ?= false
USER_NAMESPACE ?= kubeflow-user-example-com
UPLOAD_K8S_CLIENT ?= false
TLS_ENABLED ?= false
CA_CERT_PATH ?=
PULL_NUMBER ?=
REPO_NAME ?=
K8S_VERSION ?= v1.34.0
CLUSTER_NAME ?= kfp

GINKGO_FLAGS ?=
ifeq ($(GITHUB_ACTIONS),true)
	GINKGO_FLAGS += --github-output=true
endif

.PHONY: test-e2e
test-e2e:
	cd $(E2E_TEST_DIR) && \
	go run github.com/onsi/ginkgo/v2/ginkgo -r -v --cover -p --keep-going $(GINKGO_FLAGS) --nodes=$(NUM_NODES) --label-filter=$(LABEL_FILTER) --silence-skips=true -- \
		-namespace=$(NAMESPACE) \
		-multiUserMode=$(MULTI_USER) \
		-useProxy=$(USE_PROXY) \
		-userNamespace=$(USER_NAMESPACE) \
		-uploadPipelinesWithKubernetes=$(UPLOAD_K8S_CLIENT) \
		-tlsEnabled=$(TLS_ENABLED) \
		-caCertPath=$(CA_CERT_PATH) \
		-pullNumber=$(PULL_NUMBER) \
		-repoName=$(REPO_NAME)

.PHONY: build-modelcar-image
build-modelcar-image:
	docker build -f ./test_data/sdk_compiled_pipelines/valid/critical/modelcar/Dockerfile -t registry.domain.local/modelcar:test .
	kind --name $(CLUSTER_NAME) load docker-image registry.domain.local/modelcar:test

.PHONY: test-artifact-proxy
test-artifact-proxy:
	./test/artifact-proxy/test-artifact-proxy.sh "$(USER_NAMESPACE)"

.PHONY: create-kind-cluster
create-kind-cluster:
	kind create cluster --name $(CLUSTER_NAME) --image kindest/node:$(K8S_VERSION)
	kubectl cluster-info --context kind-$(CLUSTER_NAME)
