REPO_ROOT=../..

# These vars are expected in .env:
# GCS_ROOT=gs://$(PROJECT)/v2-sample-test
# GCR_ROOT=gcr.io/$(PROJECT)/v2-sample-test
# HOST=https://71a74112589c16e8-dot-asia-east1.pipelines.googleusercontent.com
ENV_PATH?=.env
include $(ENV_PATH)
SHELL = /bin/bash

.PHONY: all
all: sample-test

.PHONY: sample-test
sample-test: upload
# The -u flag makes python output unbuffered, so that we can see real time log.
# Reference: https://stackoverflow.com/a/107717
	python -u sample_test.py \
		--samples_config samples/test/config.yaml \
		--context $(GCS_ROOT)/src/context.tar.gz \
		--host $(HOST) \
		--gcs_root $(GCS_ROOT)/data \
		--gcr_root $(GCR_ROOT) \
		--kfp_package_path "$(KFP_PACKAGE_PATH)"

.PHONY: integration-test
integration-test: upload
	python -u sample_test.py \
		--samples_config samples/test/config-integration.yaml \
		--timeout_mins 60 \
		--context $(GCS_ROOT)/src/context.tar.gz \
		--host $(HOST) \
		--gcs_root $(GCS_ROOT)/data \
		--gcr_root $(GCR_ROOT) \
		--kfp_package_path "$(KFP_PACKAGE_PATH)"

.PHONY: upload
upload: context
	python -u scripts/upload_gcs_blob.py tmp/context.tar.gz $(GCS_ROOT)/src/context.tar.gz

.PHONY: context
context:
	# Package source dir into a context.tar.gz.
	mkdir -p tmp
# Benefit of using `git archive` over `tar` is that, `--exclude-vcs-ignore` flag
# we want to use is not supported in MacOS (BSD tar). This feature is only
# available in GNU tar.
#
# The following line is implemented based on the idea referenced below that can
# archive source folder taking .gitignore into account. This command works no
# matter your working dir is dirty or not. It always packages current content
# into the tarball.
# Reference: https://stackoverflow.com/a/23486703/8745218
#
# Note, there's one caveat, for any files not tracked by git, they will not be uploaded.
# So recommend doing a `git add -A` before running this if you added new files. However,
# it's OK to have dirty files, the dirty version in your workdir will be uploaded
# as expected.
	cd $(REPO_ROOT); \
	stash=$$(git stash create); \
	git archive --format=tar "$${stash:-HEAD}" | gzip >v2/test/tmp/context.tar.gz

.PHONY: mlmd-port-forward
mlmd-port-forward:
	kubectl port-forward svc/metadata-grpc-service 8080:8080
