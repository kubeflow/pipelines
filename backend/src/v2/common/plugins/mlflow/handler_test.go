package mlflow

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang/glog"
	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/plugins/mlflow"
	"github.com/kubeflow/pipelines/backend/src/v2/common/plugins"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var taskInfoStart = &plugins.TaskInfo{
	Name: "test-task",
}

var taskInfoEnd = &plugins.TaskInfo{
	Name:       "test-task",
	RunEndTime: int64(1714400000000),
	RunStatus:  "COMPLETED",
	ScalarMetrics: map[string]float64{
		"test-metric": 0.5,
	},
	Parameters: map[string]string{
		"test-param": "test-value",
	},
}

var taskStartResult = &MLflowStartResult{
	RunID: "test-run-id",
}

var emptyTaskStartResult = &MLflowStartResult{}

// setupTestEnvWithServer sets up a test MLflow server using the given HTTP handler and configures runtime settings for testing.
func setupTestEnvWithServer(t *testing.T, httpHandler http.Handler) {
	t.Helper()
	setupSAToken(t)
	server := httptest.NewServer(httpHandler)
	t.Cleanup(server.Close)
	setRuntimeCfg(commonmlflow.MLflowRuntimeConfig{
		Endpoint:     server.URL,
		ParentRunID:  "test-parent-run-id",
		ExperimentID: "test-exp",
		AuthType:     "kubernetes",
		Timeout:      "10s",
	})
}

func setupSAToken(t *testing.T) func() {
	t.Helper()
	setupFakeKubernetesConfig(t, "test-sa-token")
	return func() {} // cleanup handled by t.Cleanup in setupFakeKubernetesConfig
}

func setRuntimeCfg(runtimeCfg commonmlflow.MLflowRuntimeConfig) {
	data, err := json.Marshal(runtimeCfg)
	if err != nil {
		glog.Fatalf("Failed to marshal MLflow runtime config: %v", err)
	}
	viper.Set(commonmlflow.EnvMLflowConfig, string(data))
}

// setupFakeKubernetesConfig writes a temp kubeconfig with the given bearer token
// and sets the KUBECONFIG env var so util.GetKubernetesConfig() picks it up.
func setupFakeKubernetesConfig(t *testing.T, token string) {
	t.Helper()
	kubeconfig := fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://localhost
  name: test
contexts:
- context:
    cluster: test
    user: test
  name: test
current-context: test
users:
- name: test
  user:
    token: %s
`, token)
	p := filepath.Join(t.TempDir(), "kubeconfig")
	require.NoError(t, os.WriteFile(p, []byte(kubeconfig), 0600))
	t.Setenv("KUBECONFIG", p)
}

// defaultMLflowHandlerFunc provides a default HTTP handler for testing MLflow server interactions.
func defaultMLflowHandlerFunc(t *testing.T) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/2.0/mlflow/runs/create":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"run":{"info":{"run_id":"mlflow-run-1"}}}`))
		case "/api/2.0/mlflow/runs/update":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"run":{"info":{"run_id":"mlflow-run-1"}}}`))
		case "/api/2.0/mlflow/runs/log-batch":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	})
}

func TestNewMLflowTaskHandler_Success(t *testing.T) {
	setRuntimeCfg(commonmlflow.MLflowRuntimeConfig{
		Endpoint:     "http://localhost",
		ParentRunID:  "test-parent-run-id",
		ExperimentID: "test-exp",
		AuthType:     "kubernetes",
		Timeout:      "10s",
	})

	handler, err := NewMLflowTaskHandler()

	require.NoError(t, err)
	require.NotNil(t, handler)
}

func TestNewMLflowTaskHandler_NoEnvVar_Failure(t *testing.T) {
	viper.Set(commonmlflow.EnvMLflowConfig, "")

	handler, err := NewMLflowTaskHandler()

	require.Nil(t, handler)
	require.Error(t, err)
	assert.Equal(t, "failed to parse MLflow runtime config: KFP_MLFLOW_CONFIG env var not set", err.Error())
}

func TestNewMLflowTaskHandler_EmptyRuntimeConfig_Failure(t *testing.T) {
	setRuntimeCfg(commonmlflow.MLflowRuntimeConfig{})

	handler, err := NewMLflowTaskHandler()

	require.Nil(t, handler)
	require.Error(t, err)
	assert.Equal(t, "failed to parse MLflow runtime config: unsupported auth type: ", err.Error())
}

func TestNewMLflowTaskHandler_InvalidAuthType_Failure(t *testing.T) {
	setRuntimeCfg(commonmlflow.MLflowRuntimeConfig{
		Endpoint:     "http://localhost",
		ParentRunID:  "test-parent-run-id",
		ExperimentID: "test-exp",
		AuthType:     "invalid-auth-type",
		Timeout:      "10s",
	})

	handler, err := NewMLflowTaskHandler()
	require.Nil(t, handler)
	require.Error(t, err)
	assert.Equal(t, "failed to parse MLflow runtime config: unsupported auth type: invalid-auth-type", err.Error())
}

func TestOnTaskStart_MissingParentRunID_Failure(t *testing.T) {
	setRuntimeCfg(
		commonmlflow.MLflowRuntimeConfig{
			Endpoint:     "http://localhost",
			ExperimentID: "test-exp",
			AuthType:     "kubernetes",
			Timeout:      "10s",
		})

	handler, _ := NewMLflowTaskHandler()
	nestedRunID, err := handler.OnTaskStart(context.Background(), taskInfoStart)
	assert.Empty(t, nestedRunID)

	require.Error(t, err)
	assert.Equal(t, "ParentRunID and ExperimentID are both required to create nested MLflow run", err.Error())
}

func TestOnTaskStart_MissingExperimentID_Failure(t *testing.T) {
	setRuntimeCfg(
		commonmlflow.MLflowRuntimeConfig{
			Endpoint:    "http://localhost",
			ParentRunID: "test-parent-run-id",
			AuthType:    "kubernetes",
			Timeout:     "10s",
		})

	handler, _ := NewMLflowTaskHandler()
	nestedRunID, err := handler.OnTaskStart(context.Background(), taskInfoStart)
	assert.Empty(t, nestedRunID)

	require.Error(t, err)
	assert.Equal(t, "ParentRunID and ExperimentID are both required to create nested MLflow run", err.Error())
}

func TestOnTaskStart_Success(t *testing.T) {
	setupTestEnvWithServer(t, defaultMLflowHandlerFunc(t))

	handler, _ := NewMLflowTaskHandler()

	nestedRunID, err := handler.OnTaskStart(context.Background(), taskInfoStart)
	require.NoError(t, err)
	require.NotEmpty(t, nestedRunID)

}

func TestOnTaskStart_MLflowFailure_ReturnsEmptyNestedRunID(t *testing.T) {
	cleanup := setupSAToken(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error_code":"INTERNAL_ERROR","message":"server down"}`))
	}))
	defer server.Close()

	setRuntimeCfg(commonmlflow.MLflowRuntimeConfig{
		Endpoint:     server.URL,
		ParentRunID:  "test-parent-run-id",
		ExperimentID: "test-exp",
		AuthType:     "kubernetes",
		Timeout:      "10s",
	})
	handler, _ := NewMLflowTaskHandler()

	nestedRunID, err := handler.OnTaskStart(context.Background(), taskInfoStart)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create task-level MLflow run: ")
	require.Empty(t, nestedRunID)
}

func TestOnTaskEnd_InputRunID_Success(t *testing.T) {
	viper.Set("MLFLOW_RUN_ID", "")
	setupTestEnvWithServer(t, defaultMLflowHandlerFunc(t))

	handler, _ := NewMLflowTaskHandler()
	err := handler.OnTaskEnd(context.Background(), taskStartResult, taskInfoEnd)

	require.NoError(t, err)
}

func TestOnTaskEnd_EnvVarRunID_Success(t *testing.T) {
	viper.Set("MLFLOW_RUN_ID", "test-run-id")
	setupTestEnvWithServer(t, defaultMLflowHandlerFunc(t))

	handler, _ := NewMLflowTaskHandler()
	err := handler.OnTaskEnd(context.Background(), taskStartResult, taskInfoEnd)

	require.NoError(t, err)
}

// OnTaskEnd() will prioritize input runID over env var runID
func TestOnTaskEnd_InputAndEnvVarRunID_Success(t *testing.T) {
	viper.Set("MLFLOW_RUN_ID", "env-test-run-id")
	cleanup := setupSAToken(t)
	defer cleanup()
	var capturedUpdateBody map[string]interface{}
	var capturedLogBatchBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/2.0/mlflow/runs/update":
			require.NoError(t, json.NewDecoder(r.Body).Decode(&capturedUpdateBody))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"run":{"info":{"run_id":"test-run-id"}}}`))
		case "/api/2.0/mlflow/runs/log-batch":
			require.NoError(t, json.NewDecoder(r.Body).Decode(&capturedLogBatchBody))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	setRuntimeCfg(commonmlflow.MLflowRuntimeConfig{
		Endpoint:     server.URL,
		ParentRunID:  "test-parent-run-id",
		ExperimentID: "test-exp",
		AuthType:     "kubernetes",
		Timeout:      "10s",
	})

	handler, _ := NewMLflowTaskHandler()
	err := handler.OnTaskEnd(context.Background(), taskStartResult, taskInfoEnd)

	require.NoError(t, err)
	assert.Equal(t, "test-run-id", capturedUpdateBody["run_id"])
	assert.Equal(t, "test-run-id", capturedLogBatchBody["run_id"])
}

func TestOnTaskEnd_MissingRunID_Failure(t *testing.T) {
	viper.Set("MLFLOW_RUN_ID", "")
	setRuntimeCfg(commonmlflow.MLflowRuntimeConfig{
		Endpoint:     "http://localhost",
		ParentRunID:  "test-parent-run-id",
		ExperimentID: "test-exp",
		AuthType:     "kubernetes",
		Timeout:      "10s",
	})

	handler, _ := NewMLflowTaskHandler()
	err := handler.OnTaskEnd(context.Background(), emptyTaskStartResult, taskInfoEnd)

	require.Error(t, err)
	assert.Equal(t, "runID is required to update nested MLflow run", err.Error())
}

func TestOnTaskEnd_MissingExperimentID_Failure(t *testing.T) {
	setRuntimeCfg(commonmlflow.MLflowRuntimeConfig{
		Endpoint:    "http://localhost",
		ParentRunID: "test-parent-run-id",
		AuthType:    "kubernetes",
		Timeout:     "10s",
	})

	handler, _ := NewMLflowTaskHandler()
	err := handler.OnTaskEnd(context.Background(), taskStartResult, taskInfoEnd)

	require.Error(t, err)
	assert.Equal(t, "experimentID is required to update nested MLflow run", err.Error())
}

func TestOnTaskEnd_EmptyMetrics_Success(t *testing.T) {
	setupTestEnvWithServer(t, defaultMLflowHandlerFunc(t))
	info := &plugins.TaskInfo{
		Name:          "test-task",
		RunEndTime:    int64(1714400000000),
		RunStatus:     "COMPLETED",
		ScalarMetrics: map[string]float64{},
		Parameters: map[string]string{
			"test-param": "test-value",
		},
	}

	handler, _ := NewMLflowTaskHandler()
	err := handler.OnTaskEnd(context.Background(), taskStartResult, info)

	require.NoError(t, err)
}

func TestOnTaskEnd_NilMetrics_Success(t *testing.T) {
	setupTestEnvWithServer(t, defaultMLflowHandlerFunc(t))
	info := &plugins.TaskInfo{
		Name:          "test-task",
		RunEndTime:    int64(1714400000000),
		RunStatus:     "COMPLETED",
		ScalarMetrics: nil,
		Parameters: map[string]string{
			"test-param": "test-value",
		},
	}

	handler, _ := NewMLflowTaskHandler()
	err := handler.OnTaskEnd(context.Background(), taskStartResult, info)

	require.NoError(t, err)
}

func TestOnTaskEnd_EmptyParams_Success(t *testing.T) {
	setupTestEnvWithServer(t, defaultMLflowHandlerFunc(t))
	info := &plugins.TaskInfo{
		Name:       "test-task",
		RunEndTime: int64(1714400000000),
		RunStatus:  "COMPLETED",
		ScalarMetrics: map[string]float64{
			"test-metric": 0.5,
		},
		Parameters: map[string]string{},
	}

	handler, _ := NewMLflowTaskHandler()
	err := handler.OnTaskEnd(context.Background(), taskStartResult, info)

	require.NoError(t, err)
}

func TestOnTaskEnd_NilParams_Success(t *testing.T) {
	setupTestEnvWithServer(t, defaultMLflowHandlerFunc(t))
	info := &plugins.TaskInfo{
		Name:       "test-task",
		RunEndTime: int64(1714400000000),
		RunStatus:  "COMPLETED",
		ScalarMetrics: map[string]float64{
			"test-metric": 0.5,
		},
		Parameters: nil,
	}

	handler, _ := NewMLflowTaskHandler()
	err := handler.OnTaskEnd(context.Background(), taskStartResult, info)

	require.NoError(t, err)
}

func TestOnTaskEnd_Success(t *testing.T) {
	setupTestEnvWithServer(t, defaultMLflowHandlerFunc(t))

	handler, _ := NewMLflowTaskHandler()
	err := handler.OnTaskEnd(context.Background(), taskStartResult, taskInfoEnd)
	require.NoError(t, err)
}

func TestOnTaskStart_MLflowFailure_ReturnsError(t *testing.T) {
	cleanup := setupSAToken(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error_code":"INTERNAL_ERROR","message":"server down"}`))
	}))
	defer server.Close()

	setRuntimeCfg(commonmlflow.MLflowRuntimeConfig{
		Endpoint:     server.URL,
		ParentRunID:  "test-parent-run-id",
		ExperimentID: "test-exp",
		AuthType:     "kubernetes",
		Timeout:      "10s",
	})

	handler, _ := NewMLflowTaskHandler()
	err := handler.OnTaskEnd(context.Background(), taskStartResult, taskInfoEnd)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update MLflow run")
}

func TestRetrieveUserContainerEnvVars_WorkspacesEnabled_InjectVars_Success(t *testing.T) {
	setRuntimeCfg(commonmlflow.MLflowRuntimeConfig{
		Endpoint:          "http://localhost",
		Workspace:         "test-workspace",
		WorkspacesEnabled: true,
		ParentRunID:       "test-parent-run-id",
		ExperimentID:      "test-exp",
		AuthType:          "kubernetes",
		Timeout:           "10s",
		InjectUserEnvVars: true,
	})

	expectedEnvVars := map[string]string{
		"MLFLOW_RUN_ID":        "test-run-id",
		"MLFLOW_TRACKING_URI":  "http://localhost",
		"MLFLOW_EXPERIMENT_ID": "test-exp",
		"MLFLOW_WORKSPACE":     "test-workspace",
		"MLFLOW_TRACKING_AUTH": "kubernetes-namespaced",
	}

	handler, _ := NewMLflowTaskHandler()
	envVars, err := handler.RetrieveUserContainerEnvVars(taskStartResult)

	require.NoError(t, err)
	assert.ElementsMatch(t, expectedEnvVars, envVars)
}

func TestRetrieveUserContainerEnvVars_WorkspacesDisabled_InjectVars_Success(t *testing.T) {
	setRuntimeCfg(commonmlflow.MLflowRuntimeConfig{
		Endpoint:           "http://localhost",
		WorkspacesEnabled:  false,
		ParentRunID:        "test-parent-run-id",
		ExperimentID:       "test-exp",
		AuthType:           "kubernetes",
		Timeout:            "10s",
		InsecureSkipVerify: true,
		InjectUserEnvVars:  true,
	})

	expectedEnvVars := map[string]string{
		"MLFLOW_RUN_ID":        "test-run-id",
		"MLFLOW_TRACKING_URI":  "http://localhost",
		"MLFLOW_EXPERIMENT_ID": "test-exp",
		"MLFLOW_TRACKING_AUTH": "kubernetes-namespaced",
	}

	handler, _ := NewMLflowTaskHandler()
	envVars, err := handler.RetrieveUserContainerEnvVars(taskStartResult)

	require.NoError(t, err)
	assert.ElementsMatch(t, expectedEnvVars, envVars)
}

func TestRetrieveUserContainerEnvVars_EmptyRunID_Success(t *testing.T) {
	viper.Set("MLFLOW_RUN_ID", "")
	setRuntimeCfg(commonmlflow.MLflowRuntimeConfig{
		Endpoint:           "http://localhost",
		Workspace:          "test-workspace",
		WorkspacesEnabled:  true,
		ParentRunID:        "test-parent-run-id",
		ExperimentID:       "test-exp",
		AuthType:           "kubernetes",
		Timeout:            "10s",
		InsecureSkipVerify: true,
		InjectUserEnvVars:  true,
	})

	expectedEnvVars := map[string]string{
		"MLFLOW_TRACKING_URI":  "http://localhost",
		"MLFLOW_EXPERIMENT_ID": "test-exp",
		"MLFLOW_WORKSPACE":     "test-workspace",
		"MLFLOW_TRACKING_AUTH": "kubernetes-namespaced",
	}

	handler, _ := NewMLflowTaskHandler()
	envVars, err := handler.RetrieveUserContainerEnvVars(emptyTaskStartResult)

	require.NoError(t, err)
	assert.ElementsMatch(t, expectedEnvVars, envVars)
}

func TestRetrieveUserContainerEnvVars_InjectVarsDisabled_Success(t *testing.T) {
	setRuntimeCfg(commonmlflow.MLflowRuntimeConfig{
		Endpoint:           "http://localhost",
		Workspace:          "test-workspace",
		WorkspacesEnabled:  true,
		ParentRunID:        "test-parent-run-id",
		ExperimentID:       "test-exp",
		AuthType:           "kubernetes",
		Timeout:            "10s",
		InsecureSkipVerify: true,
		InjectUserEnvVars:  false,
	})

	expectedEnvVars := map[string]string{
		"MLFLOW_RUN_ID": "test-run-id",
	}

	handler, _ := NewMLflowTaskHandler()
	envVars, err := handler.RetrieveUserContainerEnvVars(taskStartResult)

	require.NoError(t, err)
	assert.ElementsMatch(t, expectedEnvVars, envVars)
}

func TestRetrieveCustomProperties_WithRunID(t *testing.T) {
	setRuntimeCfg(commonmlflow.MLflowRuntimeConfig{
		Endpoint:     "http://localhost",
		ParentRunID:  "test-parent-run-id",
		ExperimentID: "test-exp",
		AuthType:     "kubernetes",
		Timeout:      "10s",
	})

	handler, _ := NewMLflowTaskHandler()
	props := handler.RetrieveCustomProperties(taskStartResult)

	assert.Equal(t, map[string]string{"plugins.mlflow.run_id": "test-run-id"}, props)
}

func TestRetrieveCustomProperties_EmptyRunID(t *testing.T) {
	setRuntimeCfg(commonmlflow.MLflowRuntimeConfig{
		Endpoint:     "http://localhost",
		ParentRunID:  "test-parent-run-id",
		ExperimentID: "test-exp",
		AuthType:     "kubernetes",
		Timeout:      "10s",
	})

	handler, _ := NewMLflowTaskHandler()
	props := handler.RetrieveCustomProperties(emptyTaskStartResult)

	assert.Nil(t, props)
}

func TestRetrieveCustomProperties_NilResult(t *testing.T) {
	setRuntimeCfg(commonmlflow.MLflowRuntimeConfig{
		Endpoint:     "http://localhost",
		ParentRunID:  "test-parent-run-id",
		ExperimentID: "test-exp",
		AuthType:     "kubernetes",
		Timeout:      "10s",
	})

	handler, _ := NewMLflowTaskHandler()
	props := handler.RetrieveCustomProperties(nil)

	assert.Nil(t, props)
}

func TestApplyCustomProperties_Success(t *testing.T) {
	setRuntimeCfg(commonmlflow.MLflowRuntimeConfig{
		Endpoint:     "http://localhost",
		ParentRunID:  "test-parent-run-id",
		ExperimentID: "test-exp",
		AuthType:     "kubernetes",
		Timeout:      "10s",
	})

	handler, _ := NewMLflowTaskHandler()

	testProps := map[string]string{
		"plugins.mlflow.run_id": "custom-prop-run-id",
	}
	err := handler.ApplyCustomProperties(testProps)
	assert.NoError(t, err)
	assert.Equal(t, "custom-prop-run-id", handler.runtimeCfg.ParentRunID)

}
