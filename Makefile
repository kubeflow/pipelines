BUILD=backend/build

# Whenever build command for any of the binaries change, we should update them both here and in backend/Dockerfiles.

.PHONY: all
all: license_apiserver license_persistence_agent

$(BUILD)/apiserver:
	GO111MODULE=on go build -o $(BUILD)/apiserver github.com/kubeflow/pipelines/backend/src/apiserver

.PHONY: license_apiserver
license_apiserver: $(BUILD)/apiserver
	go-licenses csv $(BUILD)/apiserver | tee third_party/license_info.csv

$(BUILD)/persistence_agent:
	GO111MODULE=on go build -o $(BUILD)/persistence_agent github.com/kubeflow/pipelines/backend/src/agent/persistence

.PHONY: license_persistence_agent
license_persistence_agent: $(BUILD)/persistence_agent
	go-licenses csv $(BUILD)/persistence_agent | tee third_party/persistence_agent/license_info.csv
