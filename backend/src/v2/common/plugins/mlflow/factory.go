package mlflow

import (
	"github.com/kubeflow/pipelines/backend/src/v2/common/plugins"
)

func init() {
	plugins.RegisterHandlerFactory(&mlflowHandlerFactory{})
}

type mlflowHandlerFactory struct{}

func (f *mlflowHandlerFactory) Name() string {
	return "MLflow"
}

func (f *mlflowHandlerFactory) IsEnabled() bool {
	return IsEnabled()
}

func (f *mlflowHandlerFactory) Create() (plugins.TaskPluginHandler, error) {
	runtimeCfg, err := ParseKfpMLflowRuntimeConfig()
	if err != nil {
		return nil, err
	}
	return NewMLflowTaskHandler(runtimeCfg)
}
