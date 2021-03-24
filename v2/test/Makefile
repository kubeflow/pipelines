PROJECT=gongyuan-pipeline-test
REPO_ROOT=../..
GCS_ROOT=gs://$(PROJECT)/sample-test/10
GCR_ROOT=gcr.io/gongyuan-pipeline-test/v2-test
HOST=https://71a74112589c16e8-dot-asia-east1.pipelines.googleusercontent.com

.PHONY: all
all: sample-test

.PHONY: sample-test
sample-test: tmp/context.tar.gz
	gsutil cp tmp/context.tar.gz $(GCS_ROOT)/src/context.tar.gz
	python sample_test.py \
		--context $(GCS_ROOT)/src/context.tar.gz \
		--host $(HOST) \
		--gcs_root $(GCS_ROOT) \
		--gcr_root $(GCR_ROOT)

.PHONY: tmp/context.tar.gz
tmp/context.tar.gz:
# Archive source folder
# ref: https://stackoverflow.com/a/12010656
# Benefit of using `git archive` over `tar` is that, `--exclude-vcs-ignore` flag
# we want to use is not supported in MacOS (bsdtar). This feature is only
# available in gnutar.
	mkdir -p tmp
	cd $(REPO_ROOT) && git archive --format=tar $$(git stash create) | gzip >v2/test/tmp/context.tar.gz
