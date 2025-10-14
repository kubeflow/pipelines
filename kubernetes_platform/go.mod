module github.com/kubeflow/pipelines/kubernetes_platform

go 1.24.6

require (
	github.com/kubeflow/pipelines/api v0.0.0-00010101000000-000000000000
	google.golang.org/protobuf v1.36.6
)

require google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect

replace github.com/kubeflow/pipelines/api => ../api
