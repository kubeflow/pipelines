module github.com/kubeflow/pipelines/kubernetes_platform

go 1.21

require google.golang.org/protobuf v1.33.0

require github.com/google/go-cmp v0.5.9 // indirect

replace (
	github.com/mattn/go-sqlite3 => github.com/mattn/go-sqlite3 v1.14.18
	google.golang.org/grpc => google.golang.org/grpc v1.56.3
)
