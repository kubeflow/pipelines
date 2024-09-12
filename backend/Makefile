BUILD=build
MOD_ROOT=..
CSV_PATH=backend/third_party_licenses

# Container Build Params
CONTAINER_ENGINE ?= docker
IMG_TAG_APISERVER         ?= apiserver
IMG_TAG_PERSISTENCEAGENT  ?= persistence-agent
IMG_TAG_CACHESERVER       ?= cache-server
IMG_TAG_SCHEDULEDWORKFLOW ?= scheduledworkflow
IMG_TAG_VIEWERCONTROLLER  ?= viewercontroller
IMG_TAG_VISUALIZATION     ?= visualization
IMG_TAG_DRIVER            ?= kfp-driver
IMG_TAG_LAUNCHER          ?= kfp-launcher

# Whenever build command for any of the binaries change, we should update them both here and in backend/Dockerfiles.

.PHONY: all
all: license_apiserver license_persistence_agent license_cache_server license_swf license_viewer license_driver license_launcher

.PHONY: clean
clean:
	rm -rf $(BUILD)

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
$(BUILD)/driver:
	GO111MODULE=on go build -o $(BUILD)/driver github.com/kubeflow/pipelines/backend/src/v2/cmd/driver
$(BUILD)/launcher:
	GO111MODULE=on go build -o $(BUILD)/launcher github.com/kubeflow/pipelines/backend/src/v2/cmd/launcher-v2

# Update licenses info after dependencies changed.
# See README.md#updating-licenses-info section for more details.
.PHONY: license_apiserver
license_apiserver: $(BUILD)/apiserver
	cd $(MOD_ROOT) && go-licenses csv ./backend/src/apiserver > $(CSV_PATH)/apiserver.csv
.PHONY: license_persistence_agent
license_persistence_agent: $(BUILD)/persistence_agent
	cd $(MOD_ROOT) && go-licenses csv ./backend/src/agent/persistence > $(CSV_PATH)/persistence_agent.csv
.PHONY: license_cache_server
license_cache_server: $(BUILD)/cache_server
	cd $(MOD_ROOT) && go-licenses csv ./backend/src/cache > $(CSV_PATH)/cache_server.csv
.PHONY: license_swf
license_swf: $(BUILD)/swf
	cd $(MOD_ROOT) && go-licenses csv ./backend/src/crd/controller/scheduledworkflow > $(CSV_PATH)/swf.csv
.PHONY: license_viewer
license_viewer: $(BUILD)/viewer
	cd $(MOD_ROOT) && go-licenses csv ./backend/src/crd/controller/viewer > $(CSV_PATH)/viewer.csv
.PHONY: license_driver
license_driver: $(BUILD)/driver
	cd $(MOD_ROOT) && go-licenses csv ./backend/src/v2/cmd/driver > $(CSV_PATH)/driver.csv
.PHONY: license_launcher
license_launcher: $(BUILD)/launcher
	cd $(MOD_ROOT) && go-licenses csv ./backend/src/v2/cmd/launcher-v2 > $(CSV_PATH)/launcher.csv

.PHONY: image_all
image_all: image_apiserver image_persistence_agent image_cache image_swf image_viewer image_visualization image_driver image_launcher

.PHONY: image_apiserver
image_apiserver:
	cd $(MOD_ROOT) && ${CONTAINER_ENGINE} build -t ${IMG_TAG_APISERVER} -f backend/Dockerfile .
.PHONY: image_persistence_agent
image_persistence_agent:
	cd $(MOD_ROOT) && ${CONTAINER_ENGINE} build -t ${IMG_TAG_PERSISTENCEAGENT} -f backend/Dockerfile.persistenceagent .
.PHONY: image_cache
image_cache:
	cd $(MOD_ROOT) && ${CONTAINER_ENGINE} build -t ${IMG_TAG_CACHESERVER} -f backend/Dockerfile.cacheserver .
.PHONY: image_swf
image_swf:
	cd $(MOD_ROOT) && ${CONTAINER_ENGINE} build -t ${IMG_TAG_SCHEDULEDWORKFLOW} -f backend/Dockerfile.scheduledworkflow .
.PHONY: image_viewer
image_viewer:
	cd $(MOD_ROOT) && ${CONTAINER_ENGINE} build -t ${IMG_TAG_VIEWERCONTROLLER} -f backend/Dockerfile.viewercontroller .
.PHONY: image_visualization
image_visualization:
	cd $(MOD_ROOT) && ${CONTAINER_ENGINE} build -t ${IMG_TAG_VISUALIZATION} -f backend/Dockerfile.visualization .
.PHONY: image_driver
image_driver:
	cd $(MOD_ROOT) && ${CONTAINER_ENGINE} build -t ${IMG_TAG_DRIVER} -f backend/Dockerfile.driver .
.PHONY: image_launcher
image_launcher:
	cd $(MOD_ROOT) && ${CONTAINER_ENGINE} build -t ${IMG_TAG_LAUNCHER} -f backend/Dockerfile.launcher .
