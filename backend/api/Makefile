# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

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