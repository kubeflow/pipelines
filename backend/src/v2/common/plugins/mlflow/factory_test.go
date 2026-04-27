package mlflow

import (
	"testing"

	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/plugins/mlflow"
	"github.com/kubeflow/pipelines/backend/src/v2/common/plugins"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMlflowHandlerFactory_Name(t *testing.T) {
	factory := &mlflowHandlerFactory{}

	assert.Equal(t, "MLflow", factory.Name())
}

func TestMlflowHandlerFactory_IsEnabled_ConfigSet(t *testing.T) {
	setRuntimeCfg(commonmlflow.MLflowRuntimeConfig{
		Endpoint: "http://localhost",
		AuthType: "kubernetes",
	})
	t.Cleanup(func() { viper.Set(commonmlflow.EnvMLflowConfig, "") })

	factory := &mlflowHandlerFactory{}

	assert.True(t, factory.IsEnabled())
}

func TestMlflowHandlerFactory_IsEnabled_ConfigUnset(t *testing.T) {
	viper.Reset()

	factory := &mlflowHandlerFactory{}

	assert.False(t, factory.IsEnabled())
}

func TestMlflowHandlerFactory_Create_Success(t *testing.T) {
	setRuntimeCfg(commonmlflow.MLflowRuntimeConfig{
		Endpoint:     "http://localhost",
		ParentRunID:  "parent-run-1",
		ExperimentID: "exp-1",
		AuthType:     "kubernetes",
		Timeout:      "10s",
	})
	t.Cleanup(func() { viper.Set(commonmlflow.EnvMLflowConfig, "") })

	factory := &mlflowHandlerFactory{}
	handler, err := factory.Create()

	require.NoError(t, err)
	require.NotNil(t, handler)
	assert.Equal(t, "MLflow", handler.Name())
}

func TestMlflowHandlerFactory_Create_MissingConfig(t *testing.T) {
	viper.Set(commonmlflow.EnvMLflowConfig, "")

	factory := &mlflowHandlerFactory{}
	handler, err := factory.Create()

	require.Error(t, err)
	assert.Nil(t, handler)
	assert.Contains(t, err.Error(), "KFP_MLFLOW_CONFIG env var not set")
}

func TestMlflowHandlerFactory_Create_InvalidAuthType(t *testing.T) {
	setRuntimeCfg(commonmlflow.MLflowRuntimeConfig{
		Endpoint: "http://localhost",
		AuthType: "oauth",
	})
	t.Cleanup(func() { viper.Set(commonmlflow.EnvMLflowConfig, "") })

	factory := &mlflowHandlerFactory{}
	handler, err := factory.Create()

	require.Error(t, err)
	assert.Nil(t, handler)
	assert.Contains(t, err.Error(), "unsupported auth type: oauth")
}

func TestInitRegistersFactory(t *testing.T) {
	t.Cleanup(plugins.ResetRegistry)

	registered := plugins.RegisteredFactories()

	var found bool
	for _, factory := range registered {
		if factory.Name() == "MLflow" {
			found = true
			break
		}
	}
	assert.True(t, found, "init() should register an MLflow factory in the global registry")
}
