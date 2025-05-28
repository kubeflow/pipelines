module github.com/kubeflow/pipelines/kubernetes_platform

go 1.23

require (
	github.com/kubeflow/pipelines/api v0.0.0-00010101000000-000000000000
	google.golang.org/protobuf v1.33.0
)

require google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect

replace (
	github.com/kubeflow/pipelines/api => ../api
	github.com/mattn/go-sqlite3 => github.com/mattn/go-sqlite3 v1.14.18
	google.golang.org/grpc => google.golang.org/grpc v1.56.3
)
