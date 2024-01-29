module github.com/kubeflow/pipelines/kubernetes_platform

go 1.16

require (
	github.com/google/go-cmp v0.5.9 // indirect
	google.golang.org/protobuf v1.30.0

)

replace (
	github.com/mattn/go-sqlite3 => github.com/mattn/go-sqlite3 v1.14.18
	golang.org/x/net => golang.org/x/net v0.17.0
	google.golang.org/grpc => google.golang.org/grpc v1.56.3
)
