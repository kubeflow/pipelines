SHELL := /bin/bash

include ./Makefile.setup.mk

.PHONY: setup-python
setup-python:
	python3 -m venv .venv && \
	source .venv/bin/activate

.PHONY: test-go-unittest
test-go-unittest:
	go test -v -cover ./backend/...

.PHONY: test-backend-test
test-backend-test:
	source .venv/bin/activate && \
	TEST_SCRIPT="test-flip-coin.sh" ./.github/resources/scripts/e2e-test.sh && \
	TEST_SCRIPT="test-static-loop.sh" ./.github/resources/scripts/e2e-test.sh && \
	TEST_SCRIPT="test-dynamic-loop.sh" ./.github/resources/scripts/e2e-test.sh && \
	TEST_SCRIPT="test-env.sh" ./.github/resources/scripts/e2e-test.sh && \
	TEST_SCRIPT="test-volume.sh" ./.github/resources/scripts/e2e-test.sh

.PHONY: test-backend-test-flip-coin
test-backend-test-flip-coin:
	source .venv/bin/activate && \
	TEST_SCRIPT="test-flip-coin.sh" && source ./.github/resources/scripts/e2e-test.sh

.PHONY: test-backend-test-static-loop
test-backend-test-static-loop:
	source .venv/bin/activate && \
	TEST_SCRIPT="test-static-loop.sh" && source ./.github/resources/scripts/e2e-test.sh

.PHONY: test-backend-test-dynamic-loop
test-backend-test-dynamic-loop:
	source .venv/bin/activate && \
	TEST_SCRIPT="test-dynamic-loop.sh" && source ./.github/resources/scripts/e2e-test.sh

.PHONY: test-backend-test-env
test-backend-test-env:
	source .venv/bin/activate && \
	TEST_SCRIPT="test-env.sh" && source ./.github/resources/scripts/e2e-test.sh

.PHONY: test-backend-test-volume
test-backend-test-volume:
	source .venv/bin/activate && \
	TEST_SCRIPT="test-volume.sh" && source ./.github/resources/scripts/e2e-test.sh

.PHONY: test-backend-visualization-test
test-backend-visualization-test:
	./test/presubmit-backend-visualization.sh

.PHONY: test-e2e-initialization-tests-v1
test-e2e-initialization-tests-v1:
	./.github/resources/scripts/forward-port.sh "kubeflow" "ml-pipeline" 8888 8888 && \
	cd ./backend/test/initialization && \
	go test -v ./... -namespace kubeflow -args -runIntegrationTests=true

.PHONY: test-e2e-initialization-tests-v2
test-e2e-initialization-tests-v2:
	./.github/resources/scripts/forward-port.sh "kubeflow" "ml-pipeline" 8888 8888 && \
	cd ./backend/test/v2/initialization && \
	go test -v ./... -namespace kubeflow -args -runIntegrationTests=true

.PHONY: test-e2e-api-integration-tests-v1
test-e2e-api-integration-tests-v1:
	./.github/resources/scripts/forward-port.sh "kubeflow" "ml-pipeline" 8888 8888 && \
	./.github/resources/scripts/forward-port.sh "kubeflow" "mysql" 3306 3306 && \
	cd ./backend/test/integration && \
	go test -v ./... -namespace kubeflow -args -runIntegrationTests=true

.PHONY: test-e2e-api-integration-tests-v2
test-e2e-api-integration-tests-v2:
	./.github/resources/scripts/forward-port.sh "kubeflow" "ml-pipeline" 8888 8888 && \
	cd ./backend/test/v2/integration && \
	go test -v ./... -namespace kubeflow -args -runIntegrationTests=true

.PHONY: test-e2e-frontend-integration-test
test-e2e-frontend-integration-test:
	./.github/resources/scripts/forward-port.sh "kubeflow" "ml-pipeline" 8888 8888 && \
	./.github/resources/scripts/forward-port.sh "kubeflow" "ml-pipeline-ui" 3000 3000 && \
	cd ./test/frontend-integration-test && \
	docker build . -t kfp-frontend-integration-test:local && \
	docker run --net=host kfp-frontend-integration-test:local --remote-run true

.PHONY: test-e2e-basic-sample-tests
test-e2e-basic-sample-tests:
	./.github/resources/scripts/forward-port.sh "kubeflow" "ml-pipeline" 8888 8888 && \
	pip3 install -r ./test/sample-test/requirements.txt && \
	python3 ./test/sample-test/sample_test_launcher.py sample_test run_test --namespace kubeflow --test-name sequential --results-gcs-dir output && \
	python3 ./test/sample-test/sample_test_launcher.py sample_test run_test --namespace kubeflow --test-name exit_handler --expected-result failed --results-gcs-dir output

.PHONY: test-frontend
test-frontend:
	npm cache clean --force && \
	cd ./frontend && npm ci && \
	npm run test:ci

.PHONY: test-grpc-modules
test-grpc-modules:
	$(MAKE) setup-python && \
	pytest ./test/gcpc-tests/run_all_gcpc_modules.py

.PHONY: test-kfp-kubernetes-execution-tests
test-kfp-kubernetes-execution-tests:
	$(MAKE) setup-python && \
	./.github/resources/scripts/forward-port.sh "kubeflow" "ml-pipeline" 8888 8888 && \
	export KFP_ENDPOINT="http://localhost:8888" && \
	export TIMEOUT_SECONDS=2700 && \
	pytest ./test/kfp-kubernetes-execution-tests/sdk_execution_tests.py --asyncio-task-timeout $$TIMEOUT_SECONDS

.PHONY: test-kfp-kubernetes-library-test
test-kfp-kubernetes-library-test:
	./test/presubmit-test-kfp-kubernetes-library.sh

.PHONY: test-kfp-samples
test-kfp-samples:
	$(MAKE) setup-python && \
	./.github/resources/scripts/forward-port.sh "kubeflow" "ml-pipeline" 8888 8888 && \
	./backend/src/v2/test/sample-test.sh

.PHONY: test-kfp-sdk-runtime-tests
test-kfp-sdk-runtime-tests:
	$(MAKE) setup-python && \
	./test/presubmit-test-kfp-runtime-code.sh

.PHONY: test-kfp-sdk-tests
test-kfp-sdk-tests:
	./test/presubmit-tests-sdk.sh

.PHONY: test-kubeflow-pipelines-manifests
test-kubeflow-pipelines-manifests:
	./manifests/kustomize/hack/presubmit.sh

.PHONY: test-periodic-test
test-periodic-test:
	$(MAKE) setup-python && \
	nohup kubectl port-forward --namespace kubeflow svc/ml-pipeline 8888:8888 > kubectl-port-forward.log 2>&1 & \
	log_dir=$$(mktemp -d) && \
	./test/kfp-functional-test/kfp-functional-test.sh > $$log_dir/periodic_tests.txt

.PHONY: test-presubmit-backend
test-presubmit-backend:
	./test/presubmit-backend-test.sh

.PHONY: test-sdk-component-yaml
test-sdk-component-yaml:
	$(MAKE) setup-python && \
	./test/presubmit-component-yaml.sh

.PHONY: test-sdk-docformatter
test-sdk-docformatter:
	$(MAKE) setup-python && \
	./test/presubmit-docformatter-sdk.sh

.PHONY: test-sdk-execution
test-sdk-execution:
	$(MAKE) setup-python && \
	./.github/resources/scripts/forward-port.sh "kubeflow" "ml-pipeline" 8888 8888 && \
	export KFP_ENDPOINT="http://localhost:8888" && \
	export TIMEOUT_SECONDS=2700 && \
	pytest ./test/sdk-execution-tests/sdk_execution_tests.py --asyncio-task-timeout $$TIMEOUT_SECONDS

.PHONY: test-sdk-isort
test-sdk-isort:
	$(MAKE) setup-python && \
	./test/presubmit-isort-sdk.sh

.PHONY: test-sdk-upgrade
test-sdk-upgrade:
	$(MAKE) setup-python && \
	./test/presubmit-test-sdk-upgrade.sh

.PHONY: test-sdk-yapf
test-sdk-yapf:
	$(MAKE) setup-python && \
	./test/presubmit-yapf-sdk.sh

.PHONY: test-upgrade
test-upgrade:
	./.github/resources/scripts/forward-port.sh "kubeflow" "ml-pipeline" 8888 8888 && \
	cd backend/test/integration && \
	go test -v ./... -namespace kubeflow -args -runUpgradeTests=true -testify.m=Prepare && \
	go test -v ./... -namespace kubeflow -args -runUpgradeTests=true -testify.m=Verify && \
	cd ../v2/integration && \
	go test -v ./... -namespace kubeflow -args -runUpgradeTests=true -testify.m=Prepare && \
	go test -v ./... -namespace kubeflow -args -runUpgradeTests=true -testify.m=Verify

# Check diff for generated files
.PHONY: check-diff
check-diff:
	/bin/bash -c 'if [[ -n "$$(git status --porcelain)" ]]; then \
		echo "ERROR: Generated files are out of date"; \
		echo "Please regenerate using make clean all for api and kubernetes_platform"; \
		echo "Changes found in the following files:"; \
		git status; \
		echo "Diff of changes:"; \
		git diff; \
		exit 1; \
	fi'
