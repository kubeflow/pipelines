REPO_ROOT=../../../..

update: update_manifests update_backend update_tests update_docs

# Update all remote Git references to use the version specified in third_party/argo/VERSION file
update_manifests:
	@echo "Updating Argo Workflows version references in manifests..."
	@VERSION=$$(cat $(REPO_ROOT)/third_party/argo/VERSION); \
	echo "Updating go depedency..."; \
	sed -i.bak -E "s|github.com/argoproj/argo-workflows/v3 v[0-9]+\.[0-9]+\.[0-9]+|github.com/argoproj/argo-workflows/v3 $$VERSION|g" $(REPO_ROOT)/go.mod; \
	go mod tidy; \
	echo "Using Argo Workflows version: $$VERSION"; \
	echo "Updating Manifests..."; \
	sed -i.bak -E "s|ref=v[0-9]+\.[0-9]+\.[0-9]+|ref=$$VERSION|g" base/kustomization.yaml && \
	sed -i.bak -E "s|quay.io/argoproj/argoexec:v[0-9]+\.[0-9]+\.[0-9]+|quay.io/argoproj/argoexec:$$VERSION|g"  base/workflow-controller-deployment-patch.yaml && \
	sed -i.bak -E "s|quay.io/argoproj/workflow-controller:v[0-9]+\.[0-9]+\.[0-9]+|quay.io/argoproj/workflow-controller:$$VERSION|g"  base/workflow-controller-deployment-patch.yaml && \
	sed -i.bak -E "s|https://github.com/argoproj/argo-workflows/blob/v[0-9]+\.[0-9]+\.[0-9]+|https://github.com/argoproj/argo-workflows/blob/$$VERSION|g"  base/workflow-controller-configmap-patch.yaml && \
	sed -i.bak -E "s|ref=v[0-9]+\.[0-9]+\.[0-9]+|ref=$$VERSION|g" installs/namespace/kustomization.yaml && \
	sed -i.bak -E "s|ref=v[0-9]+\.[0-9]+\.[0-9]+|ref=$$VERSION|g" installs/namespace/cluster-scoped/kustomization.yaml && \
	sed -i.bak -E "s|ref=v[0-9]+\.[0-9]+\.[0-9]+|ref=$$VERSION|g" installs/cluster/kustomization.yaml && \
	sed -i.bak -E "s|https://github.com/argoproj/argo-workflows/blob/v[0-9]+\.[0-9]+\.[0-9]+|https://github.com/argoproj/argo-workflows/blob/$$VERSION|g"  ../../../gcp_marketplace/chart/kubeflow-pipelines/templates/argo.yaml && \
	echo "Cleaning up temp backups..."; \
	rm -f base/workflow-controller-deployment-patch.yaml.bak base/workflow-controller-configmap-patch.yaml.bak installs/namespace/kustomization.yaml.bak installs/namespace/cluster-scoped/kustomization.yaml.bak installs/cluster/kustomization.yaml.bak ../../../gcp_marketplace/chart/kubeflow-pipelines/templates/argo.yaml.bak; \
	echo "Successfully updated all manifest files to use $$VERSION"

update_backend:
	@echo "Updating Argo Workflows version references in backend..."
	@VERSION=$$(cat $(REPO_ROOT)/third_party/argo/VERSION); \
	echo "Using Argo Workflows version: $$VERSION"; \
	echo "Updating go depedency..."; \
	sed -i.bak -E "s|github.com/argoproj/argo-workflows/v3 v[0-9]+\.[0-9]+\.[0-9]+|github.com/argoproj/argo-workflows/v3 $$VERSION|g" $(REPO_ROOT)/go.mod; \
	go mod tidy; \
	echo "Updating code refs in backend..."; \
	sed -i.bak -E "s|ARGO_VERSION=v[0-9]+\.[0-9]+\.[0-9]+|ARGO_VERSION=$$VERSION|g" $(REPO_ROOT)/backend/Dockerfile && \
	sed -i.bak -E "s|argo-workflows/v3@v[0-9]+\.[0-9]+\.[0-9]+|argo-workflows/v3@$$VERSION|g" $(REPO_ROOT)/backend/src/common/types.go && \
	echo "Cleaning up temp backups..."; \
	rm -f $(REPO_ROOT)/go.mod.bak $(REPO_ROOT)/backend/Dockerfile.bak $(REPO_ROOT)/backend/src/common/types.go.bak; \
	echo "Successfully updated backend code refs to use $$VERSION"

update_tests:
	@echo "Updating Argo Workflows version references in tests..."
	@VERSION=$$(cat $(REPO_ROOT)/third_party/argo/VERSION); \
	echo "Using Argo Workflows version: $$VERSION"; \
	echo "Updating code refs in tests..."; \
	sed -i.bak -E "s|ARGO_VERSION=v[0-9]+\.[0-9]+\.[0-9]+|ARGO_VERSION=$$VERSION|g" $(REPO_ROOT)/test/install-argo-cli.sh && \
	sed -i.bak -E "s|quay.io/argoproj/argoexec:v[0-9]+\.[0-9]+\.[0-9]+|quay.io/argoproj/argoexec:$$VERSION|g"  $(REPO_ROOT)/test/tag_for_hosted.sh && \
	sed -i.bak -E "s|quay.io/argoproj/workflow-controller:v[0-9]+\.[0-9]+\.[0-9]+|quay.io/argoproj/workflow-controller:$$VERSION|g"  $(REPO_ROOT)/test/tag_for_hosted.sh && \
	echo "Cleaning up temp backups..."; \
	rm -f $(REPO_ROOT)/test/install-argo-cli.sh.bak $(REPO_ROOT)/test/tag_for_hosted.sh.bak; \
	echo "Successfully updated tests to use $$VERSION"

update_docs:
	@echo "Updating Argo Workflows version references in docs..."
	@VERSION=$$(cat $(REPO_ROOT)/third_party/argo/VERSION); \
	echo "Using Argo Workflows version: $$VERSION"; \
	echo "Updating docs..."; \
	sed -i.bak -E "s|ARGO_TAG=v[0-9]+\.[0-9]+\.[0-9]+|ARGO_TAG=$$VERSION|g" $(REPO_ROOT)/third_party/argo/README.md && \
	echo "Cleaning up temp backups..."; \
	rm -f $(REPO_ROOT)/third_party/argo/README.md.bak; \
	echo "Successfully updated docs to use $$VERSION"

