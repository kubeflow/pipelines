PROJECT=gongyuan-pipeline-test
REPO_ROOT=../..
SDK_ROOT=$(REPO_ROOT)/sdk/python

.PHONY: all
all: sample-test

.PHONY: sample-test
sample-test: tmp/context.tar.gz tmp/kfp.tar.gz
	gsutil cp tmp/context.tar.gz gs://$(PROJECT)/sample-test/src/context.tar.gz
	gsutil cp tmp/kfp.tar.gz gs://$(PROJECT)/sample-test/src/kfp-sdk.tar.gz
	python sample-test.py gs://$(PROJECT)/sample-test/src/context.tar.gz

.PHONY: tmp/context.tar.gz
tmp/context.tar.gz:
	mkdir -p tmp
	tar -C .. \
		--exclude=".vscode" --exclude="*.tar.gz" \
		-zcvf tmp/context.tar.gz \
		.

.PHONY: tmp/kfp.tar.gz
tmp/kfp.tar.gz:
	cd $(SDK_ROOT) && ./build.sh
	cp $(SDK_ROOT)/kfp.tar.gz tmp/kfp.tar.gz
