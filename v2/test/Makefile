PROJECT=gongyuan-pipeline-test
REPO_ROOT=../..
GCS_ROOT=gs://$(PROJECT)/sample-test/8
# HOST=https://48b1ee8166ba25d8-dot-asia-east1.pipelines.googleusercontent.com
HOST=https://71a74112589c16e8-dot-asia-east1.pipelines.googleusercontent.com

.PHONY: all
all: sample-test

.PHONY: sample-test
sample-test: tmp/context.tar.gz
	gsutil cp tmp/context.tar.gz $(GCS_ROOT)/src/context.tar.gz
	python sample_test.py --context $(GCS_ROOT)/src/context.tar.gz --host $(HOST)

.PHONY: tmp/context.tar.gz
tmp/context.tar.gz:
	mkdir -p tmp
	tar -C $(REPO_ROOT) \
		--exclude-vcs-ignores \
		-zcvf tmp/context.tar.gz \
		sdk v2
