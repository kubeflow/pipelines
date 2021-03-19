PROJECT=gongyuan-pipeline-test
REPO_ROOT=../..
GCS_ROOT=gs://$(PROJECT)/sample-test/2

.PHONY: all
all: sample-test

.PHONY: sample-test
sample-test: tmp/context.tar.gz
	gsutil cp tmp/context.tar.gz $(GCS_ROOT)/src/context.tar.gz
	python sample_test.py --context $(GCS_ROOT)/src/context.tar.gz

.PHONY: tmp/context.tar.gz
tmp/context.tar.gz:
	mkdir -p tmp
	tar -C $(REPO_ROOT) \
		-zcvf tmp/context.tar.gz \
		sdk v2
