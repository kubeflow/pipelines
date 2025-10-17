module github.com/kubeflow/pipelines/api

go 1.24.6

require (
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250715232539-7130f93afb79
	google.golang.org/protobuf v1.36.6
)

require github.com/google/go-cmp v0.6.0 // indirect

replace (
	github.com/mattn/go-sqlite3 => github.com/mattn/go-sqlite3 v1.14.18
	golang.org/x/net => golang.org/x/net v0.33.0
	google.golang.org/grpc => google.golang.org/grpc v1.56.3
)
