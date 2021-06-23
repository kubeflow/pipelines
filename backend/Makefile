BUILD=build
MOD_ROOT=..
CSV_PATH=backend/third_party_licenses

# Whenever build command for any of the binaries change, we should update them both here and in backend/Dockerfiles.

.PHONY: all
all: license_apiserver license_persistence_agent license_cache_server license_swf license_viewer

$(BUILD)/apiserver:
	GO111MODULE=on go build -o $(BUILD)/apiserver github.com/kubeflow/pipelines/backend/src/apiserver
$(BUILD)/persistence_agent:
	GO111MODULE=on go build -o $(BUILD)/persistence_agent github.com/kubeflow/pipelines/backend/src/agent/persistence
$(BUILD)/cache_server:
	GO111MODULE=on go build -o $(BUILD)/cache_server github.com/kubeflow/pipelines/backend/src/cache
$(BUILD)/swf:
	GO111MODULE=on go build -o $(BUILD)/swf github.com/kubeflow/pipelines/backend/src/crd/controller/scheduledworkflow
$(BUILD)/viewer:
	GO111MODULE=on go build -o $(BUILD)/viewer github.com/kubeflow/pipelines/backend/src/crd/controller/viewer

.PHONY: license_apiserver
license_apiserver: $(BUILD)/apiserver
	cd $(MOD_ROOT) && go-licenses csv backend/$(BUILD)/apiserver | tee $(CSV_PATH)/apiserver.csv
.PHONY: license_persistence_agent
license_persistence_agent: $(BUILD)/persistence_agent
	cd $(MOD_ROOT) && go-licenses csv backend/$(BUILD)/persistence_agent | tee $(CSV_PATH)/persistence_agent.csv
.PHONY: license_cache_server
license_cache_server: $(BUILD)/cache_server
	cd $(MOD_ROOT) && go-licenses csv backend/$(BUILD)/cache_server | tee $(CSV_PATH)/cache_server.csv
.PHONY: license_swf
license_swf: $(BUILD)/swf
	cd $(MOD_ROOT) && go-licenses csv backend/$(BUILD)/swf | tee $(CSV_PATH)/swf.csv
.PHONY: license_viewer
license_viewer: $(BUILD)/viewer
	cd $(MOD_ROOT) && go-licenses csv backend/$(BUILD)/viewer | tee $(CSV_PATH)/viewer.csv

.PHONY: image_cache
image_cache:
	docker build -t cache-server -f backend/Dockerfile.cacheserver .
