REPO_ROOT=../../../..

# Update all remote Git references to use the version specified in third_party/argo/VERSION file
update:
	@echo "Updating Argo Workflows version references..."
	@VERSION=$$(cat $(REPO_ROOT)/third_party/argo/VERSION); \
	echo "Using Argo Workflows version: $$VERSION"; \
	sed -i.bak -E "s|ref=v[0-9]+\.[0-9]+\.[0-9]+|ref=$$VERSION|g" base/kustomization.yaml && \
	sed -i.bak -E "s|ref=v[0-9]+\.[0-9]+\.[0-9]+|ref=$$VERSION|g" installs/namespace/kustomization.yaml && \
	sed -i.bak -E "s|ref=v[0-9]+\.[0-9]+\.[0-9]+|ref=$$VERSION|g" installs/namespace/cluster-scoped/kustomization.yaml && \
	sed -i.bak -E "s|ref=v[0-9]+\.[0-9]+\.[0-9]+|ref=$$VERSION|g" installs/cluster/kustomization.yaml && \
	rm -f base/kustomization.yaml.bak installs/namespace/kustomization.yaml.bak installs/namespace/cluster-scoped/kustomization.yaml.bak installs/cluster/kustomization.yaml.bak; \
	echo "Successfully updated all kustomization files to use $$VERSION"

