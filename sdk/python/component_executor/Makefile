COMPONENT_EXECUTOR_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
KFP_DIR:=$(shell dirname $(COMPONENT_EXECUTOR_DIR))/kfp

.PHONY: all
all: clean

.PHONY: build clean
clean:
	rm -rf $(COMPONENT_EXECUTOR_DIR)/kfp/component_executor/shared
	rm -rf $(COMPONENT_EXECUTOR_DIR)/*.tar.gz
	rm -rf $(COMPONENT_EXECUTOR_DIR)/*.egg-info

.PHONY: build
build:
	rsync -ax --exclude '**/*test*' $(KFP_DIR)/shared $(COMPONENT_EXECUTOR_DIR)/kfp/component_executor/
	$(eval dist_dir := $(shell mktemp -d))
	dist_dir=
	python3 setup.py sdist --format=gztar --dist-dir "$(dist_dir)"
	cp "$(dist_dir)"/*.tar.gz .
