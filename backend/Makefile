BUILD=build
MOD_ROOT=..
CSV_PATH=backend/third_party_licenses
KIND_NAME ?= dev-pipelines-api

CERT_MANAGER_VERSION ?= v1.16.2

# Container Build Params
CONTAINER_ENGINE ?= docker
# IMG_REGISTRY can be used to automatically prepend registry details. e.g. "quay.io/kubeflow/"
IMG_REGISTRY ?=
IMG_TAG_APISERVER         ?= apiserver
IMG_TAG_PERSISTENCEAGENT  ?= persistence-agent
IMG_TAG_CACHESERVER       ?= cache-server
IMG_TAG_SCHEDULEDWORKFLOW ?= scheduledworkflow
IMG_TAG_VIEWERCONTROLLER  ?= viewercontroller
IMG_TAG_VISUALIZATION     ?= visualization
IMG_TAG_DRIVER            ?= kfp-driver
IMG_TAG_LAUNCHER          ?= kfp-launcher
IMG_TAG_WEBHOOK_PROXY     ?= domain.local/kfp/webhook-proxy:latest

# Whenever build command for any of the binaries change, we should update them both here and in backend/Dockerfiles.

.PHONY: all
all: image_all

.PHONY: clean
clean:
	rm -rf $(BUILD)

.PHONY: image_all
image_all: image_apiserver image_persistence_agent image_cache image_swf image_viewer image_visualization image_driver image_launcher

.PHONY: image_apiserver
image_apiserver:
	cd $(MOD_ROOT) && ${CONTAINER_ENGINE} build --platform linux/amd64 -t ${IMG_REGISTRY}${IMG_TAG_APISERVER} -f backend/Dockerfile .
.PHONY: image_persistence_agent
image_persistence_agent:
	cd $(MOD_ROOT) && ${CONTAINER_ENGINE} build --platform linux/amd64 -t ${IMG_REGISTRY}${IMG_TAG_PERSISTENCEAGENT} -f backend/Dockerfile.persistenceagent .
.PHONY: image_cache
image_cache:
	cd $(MOD_ROOT) && ${CONTAINER_ENGINE} build --platform linux/amd64 -t ${IMG_REGISTRY}${IMG_TAG_CACHESERVER} -f backend/Dockerfile.cacheserver .
.PHONY: image_swf
image_swf:
	cd $(MOD_ROOT) && ${CONTAINER_ENGINE} build --platform linux/amd64 -t ${IMG_REGISTRY}${IMG_TAG_SCHEDULEDWORKFLOW} -f backend/Dockerfile.scheduledworkflow .
.PHONY: image_viewer
image_viewer:
	cd $(MOD_ROOT) && ${CONTAINER_ENGINE} build --platform linux/amd64 -t ${IMG_REGISTRY}${IMG_TAG_VIEWERCONTROLLER} -f backend/Dockerfile.viewercontroller .
.PHONY: image_visualization
image_visualization:
	cd $(MOD_ROOT) && ${CONTAINER_ENGINE} build --platform linux/amd64 -t ${IMG_REGISTRY}${IMG_TAG_VISUALIZATION} -f backend/Dockerfile.visualization .
.PHONY: image_driver
image_driver:
	cd $(MOD_ROOT) && ${CONTAINER_ENGINE} build --platform linux/amd64 -t ${IMG_REGISTRY}${IMG_TAG_DRIVER} -f backend/Dockerfile.driver .
.PHONY: image_driver_debug
image_driver_debug:
	cd $(MOD_ROOT) && sed -e '/RUN .*go mod download/a\
	RUN go install github.com/go-delve/delve/cmd/dlv@latest' \
	-e '/COPY .*\/bin\/driver \/bin\/driver/a\
	COPY . \/go\/src\/github.com\/kubeflow\/pipelines\
	COPY --from=builder /go/bin/dlv /bin/dlv\
	EXPOSE 2345' \
	backend/Dockerfile.driver > backend/Dockerfile.driver-debug
	cd $(MOD_ROOT) && ${CONTAINER_ENGINE} build --platform linux/amd64 --build-arg GCFLAGS="all=-N -l" -t ${IMG_REGISTRY}${IMG_TAG_DRIVER}:debug -f backend/Dockerfile.driver-debug .
.PHONY: image_launcher
image_launcher:
	cd $(MOD_ROOT) && ${CONTAINER_ENGINE} build --platform linux/amd64 -t ${IMG_REGISTRY}${IMG_TAG_LAUNCHER} -f backend/Dockerfile.launcher .

.PHONY: install-cert-manager
install-cert-manager:
	kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/$(CERT_MANAGER_VERSION)/cert-manager.yaml
	kubectl wait deployment -n cert-manager cert-manager --for condition=Available=True --timeout=180s
	kubectl wait --for=condition=Ready pod -l app.kubernetes.io/instance=cert-manager -n cert-manager --timeout=180s

.PHONY: dev-kind-cluster
dev-kind-cluster:
	${CONTAINER_ENGINE} build -t ${IMG_TAG_WEBHOOK_PROXY} -f $(CURDIR)/../tools/kind/Dockerfile.webhook-proxy $(CURDIR)/../tools/kind
	-kind create cluster --name $(KIND_NAME) --config $(CURDIR)/../tools/kind/kind-config.yaml
	kubectl config use-context kind-$(KIND_NAME)
	kind get kubeconfig --name $(KIND_NAME) > $(CURDIR)/../kubeconfig_$(KIND_NAME)
	kind --name $(KIND_NAME) load docker-image ${IMG_TAG_WEBHOOK_PROXY}
	$(MAKE) install-cert-manager
	$(MAKE) -C $(CURDIR)/src/crd/kubernetes install
	kubectl apply -k $(CURDIR)/../manifests/kustomize/cluster-scoped-resources
	kubectl wait --for condition=established --timeout=1m crd/applications.app.k8s.io
	kubectl apply -k $(CURDIR)/../manifests/kustomize/env/dev-kind
	kubectl apply -k $(CURDIR)/../manifests/kustomize/base/crds	
	kubectl apply -f $(CURDIR)/../tools/kind/webhook-proxy.yaml
	kubectl -n kubeflow wait --for condition=Available --timeout=10m deployment/mysql
	kubectl -n kubeflow wait --for condition=Available --timeout=3m deployment/metadata-grpc-deployment

.PHONY: kind-load-driver-debug
kind-load-driver-debug:
	kind --name $(KIND_NAME) load docker-image ${IMG_TAG_DRIVER}:debug

.PHONY: kind-build-and-load-driver-debug
kind-build-and-load-driver-debug: image_driver_debug kind-load-driver-debug
