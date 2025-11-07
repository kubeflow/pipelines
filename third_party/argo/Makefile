REPO_ROOT=../..
ARGO_DIR=$(REPO_ROOT)/third_party/argo
MANIFESTS_DIR=$(REPO_ROOT)/manifests/kustomize/third-party/argo
GCP_MANIFESTS_DIR=$(REPO_ROOT)/manifests/gcp_marketplace
BACKEND_CODE_DIR=$(REPO_ROOT)/backend
TESTS_DIR=$(REPO_ROOT)/test
DOCS_DIR=$(ARGO_DIR)

update: update_manifests update_backend update_tests update_docs

# Update all remote Git references to use the version specified in third_party/argo/VERSION file
update_manifests:
	@echo "Updating Argo Workflows version references in manifests..."
	@VERSION=$$(cat $(ARGO_DIR)/VERSION); \
	echo "Using Argo Workflows version: $$VERSION"; \
	echo "Updating Manifests..."; \
	sed -i.bak -E "s|ref=v[0-9]+\.[0-9]+\.[0-9]+|ref=$$VERSION|g" $(MANIFESTS_DIR)/base/kustomization.yaml && \
	sed -i.bak -E "s|quay.io/argoproj/argoexec:v[0-9]+\.[0-9]+\.[0-9]+|quay.io/argoproj/argoexec:$$VERSION|g"  $(MANIFESTS_DIR)/base/workflow-controller-deployment-patch.yaml && \
	sed -i.bak -E "s|quay.io/argoproj/workflow-controller:v[0-9]+\.[0-9]+\.[0-9]+|quay.io/argoproj/workflow-controller:$$VERSION|g"  $(MANIFESTS_DIR)/base/workflow-controller-deployment-patch.yaml && \
	sed -i.bak -E "s|https://github.com/argoproj/argo-workflows/blob/v[0-9]+\.[0-9]+\.[0-9]+|https://github.com/argoproj/argo-workflows/blob/$$VERSION|g"  $(MANIFESTS_DIR)/base/workflow-controller-configmap-patch.yaml && \
	sed -i.bak -E "s|ref=v[0-9]+\.[0-9]+\.[0-9]+|ref=$$VERSION|g" $(MANIFESTS_DIR)/installs/namespace/kustomization.yaml && \
	sed -i.bak -E "s|ref=v[0-9]+\.[0-9]+\.[0-9]+|ref=$$VERSION|g" $(MANIFESTS_DIR)/installs/namespace/cluster-scoped/kustomization.yaml && \
	sed -i.bak -E "s|ref=v[0-9]+\.[0-9]+\.[0-9]+|ref=$$VERSION|g" $(MANIFESTS_DIR)/installs/cluster/kustomization.yaml && \
	sed -i.bak -E "s|https://github.com/argoproj/argo-workflows/blob/v[0-9]+\.[0-9]+\.[0-9]+|https://github.com/argoproj/argo-workflows/blob/$$VERSION|g"  $(GCP_MANIFESTS_DIR)/chart/kubeflow-pipelines/templates/argo.yaml && \
	echo "Cleaning up temp backups..."; \
	rm -f $(MANIFESTS_DIR)/base/kustomization.yaml.bak $(MANIFESTS_DIR)/base/workflow-controller-deployment-patch.yaml.bak $(MANIFESTS_DIR)/base/workflow-controller-configmap-patch.yaml.bak $(MANIFESTS_DIR)/installs/namespace/kustomization.yaml.bak $(MANIFESTS_DIR)/installs/namespace/cluster-scoped/kustomization.yaml.bak $(MANIFESTS_DIR)/installs/cluster/kustomization.yaml.bak $(GCP_MANIFESTS_DIR)/chart/kubeflow-pipelines/templates/argo.yaml.bak; \
	echo "Successfully updated all manifest files to use $$VERSION"

update_backend:
	@echo "Updating Argo Workflows version references in backend..."
	@VERSION=$$(cat $(ARGO_DIR)/VERSION); \
	echo "Using Argo Workflows version: $$VERSION"; \
	echo "Updating go depedency..."; \
	sed -i.bak -E "s|github.com/argoproj/argo-workflows/v3 v[0-9]+\.[0-9]+\.[0-9]+|github.com/argoproj/argo-workflows/v3 $$VERSION|g" $(REPO_ROOT)/go.mod; \
	go mod tidy; \
	echo "Updating code refs in backend..."; \
	sed -i.bak -E "s|ARGO_VERSION=v[0-9]+\.[0-9]+\.[0-9]+|ARGO_VERSION=$$VERSION|g" $(BACKEND_CODE_DIR)/Dockerfile && \
	sed -i.bak -E "s|argo-workflows/v3@v[0-9]+\.[0-9]+\.[0-9]+|argo-workflows/v3@$$VERSION|g" $(BACKEND_CODE_DIR)/src/common/types.go && \
	echo "Cleaning up temp backups..."; \
	rm -f $(REPO_ROOT)/go.mod.bak $(BACKEND_CODE_DIR)/Dockerfile.bak $(BACKEND_CODE_DIR)/src/common/types.go.bak; \
	echo "Successfully updated backend code refs to use $$VERSION"

update_tests:
	@echo "Updating Argo Workflows version references in tests..."
	@VERSION=$$(cat $(ARGO_DIR)/VERSION); \
	echo "Using Argo Workflows version: $$VERSION"; \
	echo "Updating code refs in tests..."; \
	sed -i.bak -E "s|ARGO_VERSION=v[0-9]+\.[0-9]+\.[0-9]+|ARGO_VERSION=$$VERSION|g" $(TESTS_DIR)/install-argo-cli.sh && \
	sed -i.bak -E "s|quay.io/argoproj/argoexec:v[0-9]+\.[0-9]+\.[0-9]+|quay.io/argoproj/argoexec:$$VERSION|g"  $(TESTS_DIR)/tag_for_hosted.sh && \
	sed -i.bak -E "s|quay.io/argoproj/workflow-controller:v[0-9]+\.[0-9]+\.[0-9]+|quay.io/argoproj/workflow-controller:$$VERSION|g"  $(TESTS_DIR)/tag_for_hosted.sh && \
	echo "Cleaning up temp backups..."; \
	rm -f $(TESTS_DIR)/install-argo-cli.sh.bak $(TESTS_DIR)/tag_for_hosted.sh.bak; \
	echo "Successfully updated tests to use $$VERSION"

update_docs:
	@echo "Updating Argo Workflows version references in docs..."
	@VERSION=$$(cat $(ARGO_DIR)/VERSION); \
	echo "Using Argo Workflows version: $$VERSION"; \
	echo "Updating docs..."; \
	sed -i.bak -E "s|ARGO_TAG=v[0-9]+\.[0-9]+\.[0-9]+|ARGO_TAG=$$VERSION|g" $(DOCS_DIR)/UPGRADE.md && \
	echo "Cleaning up temp backups..."; \
	rm -f $(DOCS_DIR)/UPGRADE.md.bak; \
	echo "Successfully updated docs to use $$VERSION"

