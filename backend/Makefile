BUILD=backend/build

# Whenever build command for any of the binaries change, we should update them both here and in backend/Dockerfiles.

.PHONY: all
all: license_apiserver license_persistence_agent license_cache_server

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

$(BUILD)/cache_server:
	GO111MODULE=on go build -o $(BUILD)/cache_server github.com/kubeflow/pipelines/backend/src/cache

.PHONY: license_cache_server
license_cache_server: $(BUILD)/cache_server
	go-licenses csv $(BUILD)/cache_server | tee third_party/cache_server/license_info.csv

.PHONY: image_cache_server
image_cache_server:
	docker build -t cache-server -f backend/Dockerfile.cacheserver .

$(BUILD)/swf:
	GO111MODULE=on go build -o $(BUILD)/swf github.com/kubeflow/pipelines/backend/src/crd/controller/scheduledworkflow

.PHONY: license_swf
license_swf: $(BUILD)/swf
	go-licenses csv $(BUILD)/swf | tee third_party/swf/license_info.csv

$(BUILD)/viewer:
	go build -o $(BUILD)/viewer github.com/kubeflow/pipelines/backend/src/crd/controller/viewer

.PHONY: license_viewer
license_viewer: $(BUILD)/viewer
	go-licenses csv $(BUILD)/viewer | tee third_party/viewer/license_info.csv

