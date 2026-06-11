package mlflow

import (
	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/plugins/mlflow"
	"github.com/kubeflow/pipelines/backend/src/v2/common/plugins"
)

func init() {
	plugins.RegisterHandlerFactory(&mlflowHandlerFactory{})
}

type mlflowHandlerFactory struct{}

var _ plugins.HandlerFactory = (*mlflowHandlerFactory)(nil)
var _ plugins.RuntimeArgsHandlerFactory = (*mlflowHandlerFactory)(nil)

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

func (f *mlflowHandlerFactory) IsEnabledWithRuntimeArgs(runtimeArgs map[string]string) bool {
	if runtimeArgs[commonmlflow.EnvMLflowConfig] != "" {
		return true
	}
	return f.IsEnabled()
}

func (f *mlflowHandlerFactory) CreateWithRuntimeArgs(runtimeArgs map[string]string) (plugins.TaskPluginHandler, error) {
	if runtimeArgs[commonmlflow.EnvMLflowConfig] == "" {
		return f.Create()
	}
	runtimeCfg, err := ParseKfpMLflowRuntimeConfigValue(runtimeArgs[commonmlflow.EnvMLflowConfig])
	if err != nil {
		return nil, err
	}
	return NewMLflowTaskHandler(runtimeCfg)
}
