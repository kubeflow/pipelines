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
sample-test: tmp/context.tar.gz
# The -u flag makes python output unbuffered, so that we can see real time log.
# Reference: https://stackoverflow.com/a/107717
	python -u scripts/upload_gcs_blob.py tmp/context.tar.gz $(GCS_ROOT)/src/context.tar.gz
	python -u sample_test.py \
		--context $(GCS_ROOT)/src/context.tar.gz \
		--host $(HOST) \
		--gcs_root $(GCS_ROOT)/data \
		--gcr_root $(GCR_ROOT)

.PHONY: tmp/context.tar.gz
tmp/context.tar.gz:
# Archive source folder
# ref: https://stackoverflow.com/a/12010656
# Benefit of using `git archive` over `tar` is that, `--exclude-vcs-ignore` flag
# we want to use is not supported in MacOS (bsdtar). This feature is only
# available in gnutar.
	mkdir -p tmp
# cd $(REPO_ROOT) && git archive --format=tar $$(git stash create) | gzip >v2/test/tmp/context.tar.gz
# TODO(Bobgy): figure out how to allow archiving dirty region.
	cd $(REPO_ROOT) && git archive --format=tar HEAD | gzip >v2/test/tmp/context.tar.gz
