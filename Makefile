include ./Makefile.setup.mk

.PHONY: go-unittest
go-unittest:
	go test -v -cover ./backend/...

.PHONY: backend-test
backend-test: 
	. .venv/bin/activate
	TEST_SCRIPT="test-flip-coin.sh" ./.github/resources/scripts/e2e-test.sh
	TEST_SCRIPT="test-static-loop.sh" ./.github/resources/scripts/e2e-test.sh
	TEST_SCRIPT="test-dynamic-loop.sh" ./.github/resources/scripts/e2e-test.sh
	TEST_SCRIPT="test-env.sh" ./.github/resources/scripts/e2e-test.sh
	TEST_SCRIPT="test-volume.sh" ./.github/resources/scripts/e2e-test.sh

.PHONY: backend-test-flip-coin
backend-test-flip-coin: 
	. .venv/bin/activate
	TEST_SCRIPT="test-flip-coin.sh" ./.github/resources/scripts/e2e-test.sh

.PHONY: backend-test-static-loop
backend-test-static-loop: 
	. .venv/bin/activate
	TEST_SCRIPT="test-static-loop.sh" ./.github/resources/scripts/e2e-test.sh

.PHONY: backend-test-dynamic-loop
backend-test-dynamic-loop: 
	. .venv/bin/activate
	TEST_SCRIPT="test-dynamic-loop.sh" ./.github/resources/scripts/e2e-test.sh

.PHONY: backend-test-env
backend-test-env: 
	. .venv/bin/activate
	TEST_SCRIPT="test-env.sh" ./.github/resources/scripts/e2e-test.sh

.PHONY: backend-test-volume
backend-test-volume: 
	. .venv/bin/activate
	TEST_SCRIPT="test-volume.sh" ./.github/resources/scripts/e2e-test.sh

.PHONY: backend-visualization-test
backend-visualization-test: 
	./test/presubmit-backend-visualization.sh

.PHONY: e2e-initialization-tests-v1
e2e-initialization-tests-v1:
	./.github/resources/scripts/forward-port.sh "kubeflow" "ml-pipeline" 8888 8888
	cd ./backend/test/initialization
	go test -v ./... -namespace kubeflow -args -runIntegrationTests=true

.PHONY: e2e-initialization-tests-v2
e2e-initialization-tests-v2:
	./.github/resources/scripts/forward-port.sh "kubeflow" "ml-pipeline" 8888 8888
	cd ./backend/test/v2/initialization
	go test -v ./... -namespace kubeflow -args -runIntegrationTests=true

.PHONY: e2e-api-integration-tests-v1
e2e-api-integration-tests-v1:
	./.github/resources/scripts/forward-port.sh "kubeflow" "ml-pipeline" 8888 8888
	./backend/test/integration
	go test -v ./... -namespace ${NAMESPACE} -args -runIntegrationTests=true

.PHONY: e2e-api-integration-tests-v2
e2e-api-integration-tests-v2:
	./.github/resources/scripts/forward-port.sh "kubeflow" "ml-pipeline" 8888 8888
	cd ./backend/test/v2/integration
	go test -v ./... -namespace ${NAMESPACE} -args -runIntegrationTests=true

.PHONY: e2e-frontend-integration-test
e2e-frontend-integration-test:
	./.github/resources/scripts/forward-port.sh "kubeflow" "ml-pipeline" 8888 8888
	./.github/resources/scripts/forward-port.sh "kubeflow" "ml-pipeline-ui" 3000 3000
	cd ./test/frontend-integration-test
	docker build . -t kfp-frontend-integration-test:local
	docker run --net=host kfp-frontend-integration-test:local --remote-run true
	
.PHONY: e2e-basic-sample-tests
e2e-basic-sample-tests:
	./.github/resources/scripts/forward-port.sh "kubeflow" "ml-pipeline" 8888 8888
	pip3 install -r ./test/sample-test/requirements.txt
	python3 ./test/sample-test/sample_test_launcher.py sample_test run_test --namespace kubeflow --test-name sequential --results-gcs-dir output
	python3 ./test/sample-test/sample_test_launcher.py sample_test run_test --namespace kubeflow --test-name exit_handler --expected-result failed --results-gcs-dir output
	
.PHONY: frontend-test
frontend-test:
	npm cache clean --force
	cd ./frontend && npm ci
	cd ./frontend && npm run test:ci
	
.PHONY: grpc-modules-test
grpc-modules-test:
	pytest ./test/gcpc-tests/run_all_gcpc_modules.py
	
.PHONY: kfp-kubernetes-execution-tests
kfp-kubernetes-execution-tests: 
	./.github/resources/scripts/forward-port.sh "kubeflow" "ml-pipeline" 8888 8888
	export KFP_ENDPOINT="http://localhost:8888"
	export TIMEOUT_SECONDS=2700
	pytest ./test/kfp-kubernetes-execution-tests/sdk_execution_tests.py --asyncio-task-timeout $TIMEOUT_SECONDS
	
.PHONY: kfp-kubernetes-library-test
kfp-kubernetes-library-test:
	./test/presubmit-test-kfp-kubernetes-library.sh

.PHONY: kfp-samples
kfp-samples:
	./.github/resources/scripts/forward-port.sh "kubeflow" "ml-pipeline" 8888 8888
	./backend/src/v2/test/sample-test.sh

.PHONY: kfp-sdk-runtime-tests
kfp-sdk-runtime-tests:
	./test/presubmit-test-kfp-runtime-code.sh
	
.PHONY: kfp-sdk-tests
kfp-sdk-tests:
	./test/presubmit-tests-sdk.sh

.PHONY: kubeflow-pipelines-manifests
kubeflow-pipelines-manifests:
	./manifests/kustomize/hack/presubmit.sh 

.PHONY: periodic-test
periodic-test:
	nohup kubectl port-forward --namespace kubeflow svc/ml-pipeline 8888:8888 &
	log_dir=$(mktemp -d)
	./test/kfp-functional-test/kfp-functional-test.sh > $log_dir/periodic_tests.txt

.PHONY: presubmit-backend
presubmit-backend:
	./test/presubmit-backend-test.sh

.PHONY: sdk-component-yaml
sdk-component-yaml:
	./test/presubmit-component-yaml.sh
	
.PHONY: sdk-docformatter
sdk-docformatter:
	./test/presubmit-docformatter-sdk.sh

.PHONY: sdk-execution
sdk-execution:
	./.github/resources/scripts/forward-port.sh "kubeflow" "ml-pipeline" 8888 8888
	export KFP_ENDPOINT="http://localhost:8888"
	export TIMEOUT_SECONDS=2700
	pytest ./test/sdk-execution-tests/sdk_execution_tests.py --asyncio-task-timeout $TIMEOUT_SECONDS
	
.PHONY: sdk-isort
sdk-isort:
	./test/presubmit-isort-sdk.sh
	
.PHONY: sdk-upgrade
sdk-upgrade:
	./test/presubmit-test-sdk-upgrade.sh
	
.PHONY: sdk-yapf
sdk-yapf:
	./test/presubmit-yapf-sdk.sh




