all:
	protoc -I/usr/local/include -I. \
		-I${GOPATH}/src \
		-I${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		-I${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway/ \
		--go_out=plugins=grpc:. \
		*.proto
	protoc -I/usr/local/include -I. \
		-I${GOPATH}/src \
		-I${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		-I${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway/ \
		--grpc-gateway_out=logtostderr=true:. \
		*.proto
	protoc -I/usr/local/include -I. \
		-I${GOPATH}/src \
		-I${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		-I${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway/ \
		--swagger_out=logtostderr=true:. \
		*.proto
	# Instructions to install the 'swagger' command are located here:
	# https://github.com/go-swagger/go-swagger
	swagger generate client \
		-f ./job.swagger.json \
		-A job \
		--principal models.Principal \
		-c job_client \
		-m job_model
	swagger generate client \
		-f ./run.swagger.json \
		-A run \
		--principal models.Principal \
		-c run_client \
		-m run_model
	swagger generate client \
		-f ./pipeline.swagger.json \
		-A pipeline \
		--principal models.Principal \
		-c pipeline_client \
		-m pipeline_model
	go generate .