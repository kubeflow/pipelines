package mlflow

import (
	"encoding/json"
	"testing"

	"github.com/golang/glog"
	commonplugins "github.com/kubeflow/pipelines/backend/src/common/plugins"
	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/plugins/mlflow"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func setRuntimeCfg(runtimeCfg commonmlflow.MLflowRuntimeConfig) {
	data, err := json.Marshal(runtimeCfg)
	if err != nil {
		glog.Fatalf("Failed to marshal MLflow runtime config: %v", err)
	}
	viper.Set(commonmlflow.EnvMLflowConfig, string(data))
}

func TestExecutionStateToMLflowTerminalStatus(t *testing.T) {
	assert.Equal(t, "FINISHED", ExecutionStateToMLflowTerminalStatus("COMPLETE"))
	assert.Equal(t, "FINISHED", ExecutionStateToMLflowTerminalStatus("CACHED"))
	assert.Equal(t, "KILLED", ExecutionStateToMLflowTerminalStatus("CANCELED"))
	assert.Equal(t, "FAILED", ExecutionStateToMLflowTerminalStatus("FAILED"))
	assert.Equal(t, "FAILED", ExecutionStateToMLflowTerminalStatus("UNKNOWN"))
}

func TestParseKfpMLflowRuntimeConfig_Success(t *testing.T) {
	cfg := commonmlflow.MLflowRuntimeConfig{
		Endpoint:     "http://localhost",
		ParentRunID:  "test-parent-run-id",
		ExperimentID: "test-exp",
		AuthType:     "kubernetes",
		Timeout:      "10s",
	}
	expectedCfg := &commonmlflow.MLflowRuntimeConfig{
		Endpoint:           "http://localhost",
		WorkspacesEnabled:  false,
		Workspace:          "",
		ParentRunID:        "test-parent-run-id",
		ExperimentID:       "test-exp",
		AuthType:           "kubernetes",
		Timeout:            "10s",
		InsecureSkipVerify: false,
		InjectUserEnvVars:  false,
		TLS: &commonplugins.TLSConfig{
			InsecureSkipVerify: false,
		},
	}

	setRuntimeCfg(cfg)
	runtimeCfg, err := ParseKfpMLflowRuntimeConfig()

	assert.NotNil(t, runtimeCfg)
	assert.Equal(t, expectedCfg, runtimeCfg)
	assert.NoError(t, err)
}

func TestParseKfpMLflowRuntimeConfig_NoEnvVar_Failure(t *testing.T) {
	viper.Set(commonmlflow.EnvMLflowConfig, "")

	runtimeCfg, err := ParseKfpMLflowRuntimeConfig()

	assert.Nil(t, runtimeCfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "KFP_MLFLOW_CONFIG env var not set")

}

func TestParseKfpMLflowRuntimeConfig_InvalidEnvVar_Failure(t *testing.T) {
	viper.Set(commonmlflow.EnvMLflowConfig, "invalid-formatting.")

	runtimeCfg, err := ParseKfpMLflowRuntimeConfig()

	assert.Nil(t, runtimeCfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal KFP_MLFLOW_CONFIG")
}

func TestParseKfpMLflowRuntimeConfig_MissingEndpoint_Failure(t *testing.T) {
	cfg := commonmlflow.MLflowRuntimeConfig{
		ParentRunID:  "test-parent-run-id",
		ExperimentID: "test-exp",
		AuthType:     "invalid-auth-type",
		Timeout:      "10s",
	}
	setRuntimeCfg(cfg)
	runtimeCfg, err := ParseKfpMLflowRuntimeConfig()

	assert.Nil(t, runtimeCfg)
	assert.Error(t, err)
	assert.Equal(t, "missing one or more of the following required fields in KFP_MLFLOW_CONFIG: Endpoint", err.Error())
}

func TestParseKfpMLflowRuntimeConfig_MissingParentRunId_Failure(t *testing.T) {
	cfg := commonmlflow.MLflowRuntimeConfig{
		Endpoint:     "http://localhost",
		ExperimentID: "test-exp",
		AuthType:     "invalid-auth-type",
		Timeout:      "10s",
	}
	setRuntimeCfg(cfg)
	runtimeCfg, err := ParseKfpMLflowRuntimeConfig()

	assert.Nil(t, runtimeCfg)
	assert.Error(t, err)
	assert.Equal(t, "missing one or more of the following required fields in KFP_MLFLOW_CONFIG: ParentRunID", err.Error())
}

func TestParseKfpMLflowRuntimeConfig_MissingExperimentId_Failure(t *testing.T) {
	cfg := commonmlflow.MLflowRuntimeConfig{
		Endpoint:    "http://localhost",
		ParentRunID: "test-parent-run-id",
		AuthType:    "invalid-auth-type",
		Timeout:     "10s",
	}
	setRuntimeCfg(cfg)
	runtimeCfg, err := ParseKfpMLflowRuntimeConfig()

	assert.Nil(t, runtimeCfg)
	assert.Error(t, err)
	assert.Equal(t, "missing one or more of the following required fields in KFP_MLFLOW_CONFIG: ExperimentID", err.Error())
}

func TestParseKfpMLflowRuntimeConfig_MissingAuthType_Failure(t *testing.T) {
	cfg := commonmlflow.MLflowRuntimeConfig{
		Endpoint:     "http://localhost",
		ParentRunID:  "test-parent-run-id",
		ExperimentID: "test-exp",
		Timeout:      "10s",
	}
	setRuntimeCfg(cfg)
	runtimeCfg, err := ParseKfpMLflowRuntimeConfig()

	assert.Nil(t, runtimeCfg)
	assert.Error(t, err)
	assert.Equal(t, "missing one or more of the following required fields in KFP_MLFLOW_CONFIG: AuthType", err.Error())
}

func TestParseKfpMLflowRuntimeConfig_MissingTimeout_Failure(t *testing.T) {
	cfg := commonmlflow.MLflowRuntimeConfig{
		Endpoint:     "http://localhost",
		ParentRunID:  "test-parent-run-id",
		ExperimentID: "test-exp",
		AuthType:     "kubernetes",
	}
	setRuntimeCfg(cfg)
	runtimeCfg, err := ParseKfpMLflowRuntimeConfig()

	assert.Nil(t, runtimeCfg)
	assert.Error(t, err)
	assert.Equal(t, "missing one or more of the following required fields in KFP_MLFLOW_CONFIG: Timeout", err.Error())
}

func TestParseKfpMLflowRuntimeConfig_InvalidAuthType_Failure(t *testing.T) {
	cfg := commonmlflow.MLflowRuntimeConfig{
		Endpoint:     "http://localhost",
		ParentRunID:  "test-parent-run-id",
		ExperimentID: "test-exp",
		AuthType:     "invalid-auth-type",
		Timeout:      "10s",
	}
	setRuntimeCfg(cfg)
	runtimeCfg, err := ParseKfpMLflowRuntimeConfig()

	assert.Nil(t, runtimeCfg)
	assert.Error(t, err)
	assert.Equal(t, "unsupported auth type: invalid-auth-type", err.Error())
}
