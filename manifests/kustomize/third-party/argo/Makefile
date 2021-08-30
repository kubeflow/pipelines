REPO_ROOT=../../../..

update:
	rm -rf upstream
	mkdir upstream
	kpt pkg get "https://github.com/argoproj/argo-workflows.git/manifests@$$(cat $(REPO_ROOT)/third_party/argo/VERSION)" upstream/
	# Remove the pre-hydrated manifests which we do not use.
	rm upstream/manifests/*.yaml
	# Remove README.md which might be confusing here.
	rm upstream/manifests/README.md
	# Include argo license file
	curl -Lo upstream/manifests/LICENSE "https://raw.githubusercontent.com/argoproj/argo-workflows/$$(cat $(REPO_ROOT)/third_party/argo/VERSION)/LICENSE"
