module github.com/kubeflow/pipelines/third_party/ml-metadata

go 1.16

require (
	google.golang.org/grpc v1.56.3
	google.golang.org/protobuf v1.30.0
)

replace (
	github.com/mattn/go-sqlite3 => github.com/mattn/go-sqlite3 v1.14.18
	golang.org/x/net => golang.org/x/net v0.17.0
)
