package mlflow

import (
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/plugins"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMlflowHandlerFactory_Name(t *testing.T) {
	factory := &mlflowHandlerFactory{}

	assert.Equal(t, "MLflow", factory.Name())
}

func TestMlflowHandlerFactory_IsEnabled_ConfigSet(t *testing.T) {
	viper.Set("plugins.mlflow", map[string]interface{}{
		"endpoint": "http://localhost",
	})
	t.Cleanup(viper.Reset)

	factory := &mlflowHandlerFactory{}

	assert.True(t, factory.IsEnabled())
}

func TestMlflowHandlerFactory_IsEnabled_ConfigUnset(t *testing.T) {
	viper.Reset()

	factory := &mlflowHandlerFactory{}

	assert.False(t, factory.IsEnabled())
}

func TestMlflowHandlerFactory_Create_Success(t *testing.T) {
	viper.Set("plugins.mlflow", map[string]interface{}{
		"endpoint": "http://localhost",
		"timeout":  "30s",
	})
	t.Cleanup(viper.Reset)

	factory := &mlflowHandlerFactory{}
	handler, err := factory.Create()

	require.NoError(t, err)
	require.NotNil(t, handler)
	assert.Equal(t, "mlflow", handler.Name())
}

func TestMlflowHandlerFactory_Create_ConfigUnset(t *testing.T) {
	viper.Reset()

	factory := &mlflowHandlerFactory{}
	handler, err := factory.Create()

	require.NoError(t, err)
	assert.Nil(t, handler)
}

func TestMlflowHandlerFactory_Create_InvalidConfig(t *testing.T) {
	viper.Set("plugins.mlflow", "not-a-valid-json-object")
	t.Cleanup(viper.Reset)

	factory := &mlflowHandlerFactory{}
	handler, err := factory.Create()

	require.Error(t, err)
	assert.Nil(t, handler)
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
