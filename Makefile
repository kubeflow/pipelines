.PHONY: build-controller
build-controller:
	go build -o ./bin/controller ./resources/scheduledworkflow

.PHONY: build-controller-image
build-controller-image:
	docker build -t pipelines/scheduledworkflow-controller .
