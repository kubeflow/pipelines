REPO_ROOT=../..

# These vars are expected in .env:
# PROJECT=gongyuan-pipeline-test
# GCS_ROOT=gs://$(PROJECT)/v2-sample-test
# GCR_ROOT=gcr.io/$(PROJECT)/v2-sample-test
# HOST=https://71a74112589c16e8-dot-asia-east1.pipelines.googleusercontent.com
ENV_PATH?=.env
include $(ENV_PATH)

.PHONY: all
all: sample-test

.PHONY: sample-test
sample-test: context
# The -u flag makes python output unbuffered, so that we can see real time log.
# Reference: https://stackoverflow.com/a/107717
	python -u scripts/upload_gcs_blob.py tmp/context.tar.gz $(GCS_ROOT)/src/context.tar.gz
	python -u sample_test.py \
		--context $(GCS_ROOT)/src/context.tar.gz \
		--host $(HOST) \
		--gcs_root $(GCS_ROOT)/data \
		--gcr_root $(GCR_ROOT)

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
	cd $(REPO_ROOT); \
	stash=$$(git stash create); \
	git archive --format=tar "$${stash:-HEAD}" | gzip >v2/test/tmp/context.tar.gz
