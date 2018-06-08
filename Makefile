.PHONY: build-cli
build-cli:
	go build ./cmd/pipelines

.PHONY: build-controller
build-controller:
	go build ./cmd/controller

.PHONY: build-controller-image
build-controller-image:
	docker build -t pipelines/scheduledworkflow-controller \
	-f Dockerfile-scheduledworkflow-controller .
