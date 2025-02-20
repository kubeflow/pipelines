SHELL := /bin/bash

.PHONY: create-kind-cluster
create-kind-cluster:
	kind create cluster --name kfp-test-cluster --config kind-config.yaml && \
	kubectl version --client --short

.PHONY: build-images
build-images:
	./.github/resources/scripts/build-images.sh

.PHONY: deploy-kfp-tekton
deploy-kfp-tekton:
	./.github/resources/scripts/deploy-kfp-tekton.sh

.PHONY: setup-kfp-tekton
setup-kfp-tekton:
	$(MAKE) build-images && \
	$(MAKE) deploy-kfp-tekton

.PHONY: deploy-kfp
deploy-kfp:
	./.github/resources/scripts/deploy-kfp.sh

.PHONY: setup-kfp
setup-kfp:
	$(MAKE) build-images && \
	$(MAKE) deploy-kfp

.PHONY: setup-python
setup-python:
	python3 -m venv .venv && \
	source .venv/bin/activate

.PHONY: setup-backend-test
setup-backend-test:
	python3 -m venv .venv && \
	source .venv/bin/activate && \
	pip install -e sdk/python

.PHONY: setup-backend-visualization-test
setup-backend-visualization-test:
	$(MAKE) setup-python

.PHONY: setup-frontend-test
setup-frontend-test:
	npm cache clean --force && \
	cd ./frontend && npm ci

.PHONY: setup-grpc-modules-test
setup-grpc-modules-test:
	$(MAKE) setup-python && \
	sudo apt-get update && \
	sudo apt-get install protobuf-compiler -y && \
	pip3 install setuptools && \
	pip3 freeze && \
	pip3 install wheel==0.42.0 && \
	pip install sdk/python && \
	cd ./api && \
	make clean python && \
	cd .. && \
	python3 -m pip install api/v2alpha1/python && \
	pip install components/google-cloud && \
	pip install $(shell grep 'pytest==' sdk/python/requirements-dev.txt)

.PHONY: setup-kfp-kubernetes-execution-tests
setup-kfp-kubernetes-execution-tests:
	$(MAKE) setup-python && \
	sudo apt-get update && \
	sudo apt-get install protobuf-compiler -y && \
	pip3 install setuptools && \
	pip3 freeze && \
	pip3 install wheel==0.42.0 && \
	pip3 install protobuf==4.25.3 && \
	cd ./api && \
	make clean python && \
	cd .. && \
	python3 -m pip install api/v2alpha1/python && \
	cd ./kubernetes_platform && \
	make clean python && \
	cd .. && \
	pip install -e ./kubernetes_platform/python[dev] && \
	pip install -r ./test/kfp-kubernetes-execution-tests/requirements.txt

.PHONY: setup-kfp-samples
setup-kfp-samples:
	$(MAKE) setup-python && \

.PHONY: setup-kfp-sdk-runtime-tests
setup-kfp-sdk-runtime-tests:
	$(MAKE) setup-python

.PHONY: setup-kfp-sdk-tests
setup-kfp-sdk-tests:
	$(MAKE) setup-python

.PHONY: setup-periodic-test
setup-periodic-test:
	$(MAKE) setup-python

.PHONY: setup-sdk-component-yaml
setup-sdk-component-yaml:
	$(MAKE) setup-python && \
	sudo apt-get update && \
	sudo apt-get install protobuf-compiler -y && \
	pip3 install setuptools && \
	pip3 freeze && \
	pip3 install wheel==0.42.0 && \
	pip3 install protobuf==4.25.3 && \
	cd ./api && \
	make clean python && \
	cd .. && \
	python3 -m pip install api/v2alpha1/python && \
	pip install -r ./test/sdk-execution-tests/requirements.txt

.PHONY: setup-sdk-docformatter
setup-sdk-docformatter:
	$(MAKE) setup-python

.PHONY: setup-sdk-execution
setup-sdk-execution:
	$(MAKE) setup-python && \
	sudo apt-get update && \
	sudo apt-get install protobuf-compiler -y && \
	pip3 install setuptools && \
	pip3 freeze && \
	pip3 install wheel==0.42.0 && \
	pip3 install protobuf==4.25.3 && \
	cd ./api && \
	make clean python && \
	cd .. && \
	python3 -m pip install api/v2alpha1/python && \
	pip install -r ./test/sdk-execution-tests/requirements.txt

.PHONY: setup-sdk-isort
setup-sdk-isort:
	$(MAKE) setup-python

.PHONY: setup-sdk-upgrade
setup-sdk-upgrade:
	$(MAKE) setup-python

.PHONY: setup-sdk-yapf
setup-sdk-yapf:
	$(MAKE) setup-python && \
	pip install yapf