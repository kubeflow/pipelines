module github.com/kubeflow/pipelines/backend/src/v2/cmd/runner

go 1.16

require (
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b // indirect
	github.com/spf13/cobra v1.3.0 // indirect
)

replace github.com/kubeflow/pipelines/backend/src/v2/cmd/runner/cmd => ./cmd

replace github.com/kubeflow/pipelines/backend/src/v2/component => ../../component
