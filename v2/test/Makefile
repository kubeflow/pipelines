PROJECT=gongyuan-pipeline-test

.PHONY: all
all: sample-test

.PHONY: sample-test
sample-test: context.tar.gz
	gsutil cp tmp/context.tar.gz gs://$(PROJECT)/sample-test/src/context.tar.gz
	python sample-test.py gs://$(PROJECT)/sample-test/src/context.tar.gz

.PHONY: context.tar.gz
context.tar.gz:
	mkdir -p tmp
	tar -C .. \
		--exclude=".vscode" --exclude="*.tar.gz" \
		-zcvf tmp/context.tar.gz \
		.
