module github.com/kubeflow/pipelines/api

go 1.16

require (
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1
	google.golang.org/protobuf v1.33.0
)

replace (
	github.com/mattn/go-sqlite3 => github.com/mattn/go-sqlite3 v1.14.18
	golang.org/x/net => golang.org/x/net v0.33.0
	google.golang.org/grpc => google.golang.org/grpc v1.56.3
)
