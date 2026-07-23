module github.com/kubeflow/pipelines/kubernetes_platform

go 1.26

toolchain go1.26.3

require (
	github.com/kubeflow/pipelines/api v0.0.0-00010101000000-000000000000
	google.golang.org/protobuf v1.36.11
)

require google.golang.org/genproto/googleapis/rpc v0.0.0-20260401024825-9d38bb4040a9 // indirect

replace github.com/kubeflow/pipelines/api => ../api
