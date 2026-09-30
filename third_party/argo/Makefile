REPO_ROOT=../..
SYNC_SCRIPT=$(REPO_ROOT)/.github/resources/scripts/sync_argo_versions.py
export ARGO_VERSION

.PHONY: update update_ci update_tests update_manifests update_backend update_docs

# Plan every tracked change and finish Go resolution before writing any files.
update:
	@if [ -n "$$ARGO_VERSION" ]; then \
		python3 $(SYNC_SCRIPT) --scope all --version "$$ARGO_VERSION"; \
	else \
		python3 $(SYNC_SCRIPT) --scope all; \
	fi

update_ci:
	@python3 $(SYNC_SCRIPT)

update_tests: update_ci

update_manifests:
	@python3 $(SYNC_SCRIPT) --scope manifests

update_backend:
	@python3 $(SYNC_SCRIPT) --scope backend

update_docs:
	@python3 $(SYNC_SCRIPT) --scope docs
