
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

# Upgrade test targets
TESTS_DIR ?= ./backend/test/v2/api
NUMBER_OF_PARALLEL_NODES ?= 15

.PHONY: get-last-release-tag
get-last-release-tag:
	@curl -sSL -H "Accept: application/vnd.github+json" \
		"https://api.github.com/repos/kubeflow/pipelines/releases/latest" | jq -r .tag_name

.PHONY: deploy-last-release
deploy-last-release:
	@if [ -z "$(RELEASE_TAG)" ]; then \
		echo "ERROR: RELEASE_TAG is required"; \
		echo "Usage: make deploy-last-release RELEASE_TAG=<tag>"; \
		exit 1; \
	fi
	kubectl apply -k https://github.com/kubeflow/pipelines/manifests/kustomize/cluster-scoped-resources?ref=$(RELEASE_TAG)
	kubectl apply -k https://github.com/kubeflow/pipelines/manifests/kustomize/env/platform-agnostic?ref=$(RELEASE_TAG)
	@/bin/bash -c 'source "./.github/resources/scripts/helper-functions.sh" && wait_for_pods || { echo "ERROR: Deploy unsuccessful. Not all pods are running after applying release $(RELEASE_TAG)."; exit 1; }'

.PHONY: prepare-upgrade
prepare-upgrade:
	cd $(TESTS_DIR) && \
	go run github.com/onsi/ginkgo/v2/ginkgo -r -v --cover -p --keep-going --github-output=true \
		--nodes=$(NUMBER_OF_PARALLEL_NODES) --label-filter="UpgradePreparation"
