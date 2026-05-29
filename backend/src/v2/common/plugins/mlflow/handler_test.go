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

	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/plugins/mlflow"
	"github.com/kubeflow/pipelines/backend/src/v2/common/plugins"
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
	Parameters: map[string]interface{}{
		"test-param": "test-value",
	},
	Tags: map[string]string{
		"tag-key": "tag-value",
	},
}

var taskStartResult = &MLflowStartResult{
	RunID: "test-run-id",
}

var emptyTaskStartResult = &MLflowStartResult{}

// setupTestEnvWithServer sets up a test MLflow server using the given HTTP handler and configures runtime settings for testing.
func setupTestEnvWithServer(t *testing.T, httpHandler http.Handler) string {
	t.Helper()
	setupSAToken(t)
	server := httptest.NewServer(httpHandler)
	t.Cleanup(server.Close)
	//todo: can potentially remove below.
	setRuntimeCfg(commonmlflow.MLflowRuntimeConfig{
		Endpoint:     server.URL,
		ParentRunID:  "test-parent-run-id",
		ExperimentID: "test-exp",
		AuthType:     "kubernetes",
		Timeout:      "10s",
	})
	return server.URL
}

func setupSAToken(t *testing.T) func() {
	t.Helper()
	setupFakeKubernetesConfig(t, "test-sa-token")
	return func() {} // cleanup handled by t.Cleanup in setupFakeKubernetesConfig
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
	runtimeCfg := &commonmlflow.MLflowRuntimeConfig{
		Endpoint:     "http://localhost",
		ParentRunID:  "test-parent-run-id",
		ExperimentID: "test-exp",
		AuthType:     "kubernetes",
		Timeout:      "10s",
	}

	handler, err := NewMLflowTaskHandler(runtimeCfg)

	require.NoError(t, err)
	require.NotNil(t, handler)
}

func TestNewMLflowTaskHandler_EmptyRuntimeConfig_Failure(t *testing.T) {
	runtimeCfg := &commonmlflow.MLflowRuntimeConfig{}

	handler, err := NewMLflowTaskHandler(runtimeCfg)

	require.Nil(t, handler)
	require.Error(t, err)
	assert.Equal(t, "failed to parse MLflow runtime config: unsupported auth type: ", err.Error())
}

func TestNewMLflowTaskHandler_NilRuntimeConfig_Failure(t *testing.T) {
	handler, err := NewMLflowTaskHandler(nil)

	require.Nil(t, handler)
	require.Error(t, err)
	assert.Equal(t, "cfg is nil", err.Error())
}

func TestOnTaskStart_MissingParentRunID_Failure(t *testing.T) {
	runtimeCfg := &commonmlflow.MLflowRuntimeConfig{
		Endpoint:     "http://localhost",
		ExperimentID: "test-exp",
		AuthType:     "kubernetes",
		Timeout:      "10s",
	}

	handler, _ := NewMLflowTaskHandler(runtimeCfg)
	nestedRunID, err := handler.OnTaskStart(context.Background(), taskInfoStart)
	assert.Empty(t, nestedRunID)

	require.Error(t, err)
	assert.Equal(t, "ParentRunID and ExperimentID are both required to create nested MLflow run", err.Error())
}

func TestOnTaskStart_MissingExperimentID_Failure(t *testing.T) {
	runtimeCfg := &commonmlflow.MLflowRuntimeConfig{
		Endpoint:    "http://localhost",
		ParentRunID: "test-parent-run-id",
		AuthType:    "kubernetes",
		Timeout:     "10s",
	}

	handler, _ := NewMLflowTaskHandler(runtimeCfg)
	nestedRunID, err := handler.OnTaskStart(context.Background(), taskInfoStart)
	assert.Empty(t, nestedRunID)

	require.Error(t, err)
	assert.Equal(t, "ParentRunID and ExperimentID are both required to create nested MLflow run", err.Error())
}

func TestOnTaskStart_Success(t *testing.T) {
	serverURL := setupTestEnvWithServer(t, defaultMLflowHandlerFunc(t))
	runtimeCfg := &commonmlflow.MLflowRuntimeConfig{
		Endpoint:     serverURL,
		ParentRunID:  "test-parent-run-id",
		ExperimentID: "test-exp",
		AuthType:     "kubernetes",
		Timeout:      "10s",
	}
	handler, _ := NewMLflowTaskHandler(runtimeCfg)

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

	runtimeCfg := &commonmlflow.MLflowRuntimeConfig{
		Endpoint:     server.URL,
		ParentRunID:  "test-parent-run-id",
		ExperimentID: "test-exp",
		AuthType:     "kubernetes",
		Timeout:      "10s",
	}
	handler, _ := NewMLflowTaskHandler(runtimeCfg)

	nestedRunID, err := handler.OnTaskStart(context.Background(), taskInfoStart)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create task-level MLflow run: ")
	require.Empty(t, nestedRunID)
}

func TestOnTaskEnd_NestedRunID_Success(t *testing.T) {
	serverURL := setupTestEnvWithServer(t, defaultMLflowHandlerFunc(t))
	runtimeCfg := &commonmlflow.MLflowRuntimeConfig{
		Endpoint:     serverURL,
		ParentRunID:  "test-parent-run-id",
		ExperimentID: "test-exp",
		AuthType:     "kubernetes",
		Timeout:      "10s",
	}
	handler, _ := NewMLflowTaskHandler(runtimeCfg)
	handler.nestedRunID = "test-run-id"

	err := handler.OnTaskEnd(context.Background(), taskInfoEnd)

	require.NoError(t, err)
}

func TestOnTaskEnd_NestedRunIDFromApplyCustomProperties_Success(t *testing.T) {
	serverURL := setupTestEnvWithServer(t, defaultMLflowHandlerFunc(t))
	runtimeCfg := &commonmlflow.MLflowRuntimeConfig{
		Endpoint:     serverURL,
		ParentRunID:  "test-parent-run-id",
		ExperimentID: "test-exp",
		AuthType:     "kubernetes",
		Timeout:      "10s",
	}
	handler, _ := NewMLflowTaskHandler(runtimeCfg)

	err := handler.ApplyCustomProperties(map[string]string{"plugins.mlflow.run_id": "test-run-id"})
	require.NoError(t, err)

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

	handler.runtimeCfg.Endpoint = server.URL
	err = handler.OnTaskEnd(context.Background(), taskInfoEnd)

	require.NoError(t, err)
	assert.Equal(t, "test-run-id", capturedUpdateBody["run_id"])
	assert.Equal(t, "test-run-id", capturedLogBatchBody["run_id"])
	assert.Equal(t, testFormattedTags(), capturedLogBatchBody["tags"])
}

func TestOnTaskEnd_MissingRunID_Failure(t *testing.T) {
	serverURL := setupTestEnvWithServer(t, defaultMLflowHandlerFunc(t))
	runtimeCfg := &commonmlflow.MLflowRuntimeConfig{
		Endpoint:     serverURL,
		ExperimentID: "test-exp",
		AuthType:     "kubernetes",
		Timeout:      "10s",
	}
	handler, _ := NewMLflowTaskHandler(runtimeCfg)
	err := handler.OnTaskEnd(context.Background(), taskInfoEnd)

	require.Error(t, err)
	assert.Equal(t, "runID is required to update nested MLflow run", err.Error())
}

func TestOnTaskEnd_MissingExperimentID_Failure(t *testing.T) {
	serverURL := setupTestEnvWithServer(t, defaultMLflowHandlerFunc(t))
	runtimeCfg := &commonmlflow.MLflowRuntimeConfig{
		Endpoint:    serverURL,
		ParentRunID: "test-parent-run-id",
		AuthType:    "kubernetes",
		Timeout:     "10s",
	}
	handler, _ := NewMLflowTaskHandler(runtimeCfg)
	handler.nestedRunID = "test-run-id"

	err := handler.OnTaskEnd(context.Background(), taskInfoEnd)

	require.Error(t, err)
	assert.Equal(t, "experimentID is required to update nested MLflow run", err.Error())
}

func TestOnTaskEnd_EmptyMetrics_Success(t *testing.T) {
	serverURL := setupTestEnvWithServer(t, defaultMLflowHandlerFunc(t))
	runtimeCfg := &commonmlflow.MLflowRuntimeConfig{
		Endpoint:     serverURL,
		ParentRunID:  "test-parent-run-id",
		ExperimentID: "test-exp",
		AuthType:     "kubernetes",
		Timeout:      "10s",
	}
	handler, _ := NewMLflowTaskHandler(runtimeCfg)
	handler.nestedRunID = "test-run-id"

	info := &plugins.TaskInfo{
		Name:          "test-task",
		RunEndTime:    int64(1714400000000),
		RunStatus:     "COMPLETED",
		ScalarMetrics: map[string]float64{},
		Parameters: map[string]interface{}{
			"test-param": "test-value",
		},
	}

	err := handler.OnTaskEnd(context.Background(), info)

	require.NoError(t, err)
}

func TestOnTaskEnd_NilMetrics_Success(t *testing.T) {
	serverURL := setupTestEnvWithServer(t, defaultMLflowHandlerFunc(t))
	runtimeCfg := &commonmlflow.MLflowRuntimeConfig{
		Endpoint:     serverURL,
		ParentRunID:  "test-parent-run-id",
		ExperimentID: "test-exp",
		AuthType:     "kubernetes",
		Timeout:      "10s",
	}
	handler, _ := NewMLflowTaskHandler(runtimeCfg)
	handler.nestedRunID = "test-run-id"

	info := &plugins.TaskInfo{
		Name:          "test-task",
		RunEndTime:    int64(1714400000000),
		RunStatus:     "COMPLETED",
		ScalarMetrics: nil,
		Parameters: map[string]interface{}{
			"test-param": "test-value",
		},
	}
	err := handler.OnTaskEnd(context.Background(), info)

	require.NoError(t, err)
}

func TestOnTaskEnd_EmptyParams_Success(t *testing.T) {
	serverURL := setupTestEnvWithServer(t, defaultMLflowHandlerFunc(t))
	runtimeCfg := &commonmlflow.MLflowRuntimeConfig{
		Endpoint:     serverURL,
		ParentRunID:  "test-parent-run-id",
		ExperimentID: "test-exp",
		AuthType:     "kubernetes",
		Timeout:      "10s",
	}
	handler, _ := NewMLflowTaskHandler(runtimeCfg)
	handler.nestedRunID = "test-run-id"

	info := &plugins.TaskInfo{
		Name:       "test-task",
		RunEndTime: int64(1714400000000),
		RunStatus:  "COMPLETED",
		ScalarMetrics: map[string]float64{
			"test-metric": 0.5,
		},
		Parameters: map[string]interface{}{},
	}

	err := handler.OnTaskEnd(context.Background(), info)

	require.NoError(t, err)
}

func TestOnTaskEnd_NilParams_Success(t *testing.T) {
	serverURL := setupTestEnvWithServer(t, defaultMLflowHandlerFunc(t))
	runtimeCfg := &commonmlflow.MLflowRuntimeConfig{
		Endpoint:     serverURL,
		ParentRunID:  "test-parent-run-id",
		ExperimentID: "test-exp",
		AuthType:     "kubernetes",
		Timeout:      "10s",
	}
	handler, _ := NewMLflowTaskHandler(runtimeCfg)
	handler.nestedRunID = "test-run-id"

	info := &plugins.TaskInfo{
		Name:       "test-task",
		RunEndTime: int64(1714400000000),
		RunStatus:  "COMPLETED",
		ScalarMetrics: map[string]float64{
			"test-metric": 0.5,
		},
		Parameters: nil,
	}

	err := handler.OnTaskEnd(context.Background(), info)

	require.NoError(t, err)
}

func TestOnTaskEnd_Success(t *testing.T) {
	serverURL := setupTestEnvWithServer(t, defaultMLflowHandlerFunc(t))
	runtimeCfg := &commonmlflow.MLflowRuntimeConfig{
		Endpoint:     serverURL,
		ParentRunID:  "test-parent-run-id",
		ExperimentID: "test-exp",
		AuthType:     "kubernetes",
		Timeout:      "10s",
	}
	handler, _ := NewMLflowTaskHandler(runtimeCfg)
	handler.nestedRunID = "test-run-id"
	err := handler.OnTaskEnd(context.Background(), taskInfoEnd)
	require.NoError(t, err)
}

func TestOnTaskEnd_MLflowFailure_ReturnsError(t *testing.T) {
	cleanup := setupSAToken(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error_code":"INTERNAL_ERROR","message":"server down"}`))
	}))
	defer server.Close()

	runtimeCfg := &commonmlflow.MLflowRuntimeConfig{
		Endpoint:     server.URL,
		ParentRunID:  "test-parent-run-id",
		ExperimentID: "test-exp",
		AuthType:     "kubernetes",
		Timeout:      "10s",
	}

	handler, _ := NewMLflowTaskHandler(runtimeCfg)
	handler.nestedRunID = "test-run-id"
	err := handler.OnTaskEnd(context.Background(), taskInfoEnd)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update MLflow run")
}

func TestRetrieveUserContainerEnvVars_WorkspacesEnabled_InjectVars_Success(t *testing.T) {
	runtimeCfg := &commonmlflow.MLflowRuntimeConfig{
		Endpoint:          "http://localhost",
		Workspace:         "test-workspace",
		WorkspacesEnabled: true,
		ParentRunID:       "test-parent-run-id",
		ExperimentID:      "test-exp",
		AuthType:          "kubernetes",
		Timeout:           "10s",
		InjectUserEnvVars: true,
	}

	expectedEnvVars := map[string]string{
		"MLFLOW_RUN_ID":        "test-run-id",
		"MLFLOW_TRACKING_URI":  "http://localhost",
		"MLFLOW_EXPERIMENT_ID": "test-exp",
		"MLFLOW_WORKSPACE":     "test-workspace",
		"MLFLOW_TRACKING_AUTH": "kubernetes-namespaced",
	}

	handler, _ := NewMLflowTaskHandler(runtimeCfg)
	handler.nestedRunID = "test-run-id"
	envVars, err := handler.RetrieveUserContainerEnvVars()

	require.NoError(t, err)
	assert.Equal(t, expectedEnvVars, envVars)
}

func TestRetrieveUserContainerEnvVars_WorkspacesDisabled_InjectVars_Success(t *testing.T) {
	runtimeCfg := &commonmlflow.MLflowRuntimeConfig{
		Endpoint:           "http://localhost",
		WorkspacesEnabled:  false,
		ParentRunID:        "test-parent-run-id",
		ExperimentID:       "test-exp",
		AuthType:           "kubernetes",
		Timeout:            "10s",
		InsecureSkipVerify: true,
		InjectUserEnvVars:  true,
	}

	expectedEnvVars := map[string]string{
		"MLFLOW_RUN_ID":        "test-run-id",
		"MLFLOW_TRACKING_URI":  "http://localhost",
		"MLFLOW_EXPERIMENT_ID": "test-exp",
		"MLFLOW_TRACKING_AUTH": "kubernetes",
	}

	handler, _ := NewMLflowTaskHandler(runtimeCfg)
	handler.nestedRunID = "test-run-id"
	envVars, err := handler.RetrieveUserContainerEnvVars()

	require.NoError(t, err)
	assert.Equal(t, expectedEnvVars, envVars)
}

func TestRetrieveUserContainerEnvVars_EmptyRunID_Failure(t *testing.T) {
	runtimeCfg := &commonmlflow.MLflowRuntimeConfig{
		Endpoint:           "http://localhost",
		Workspace:          "test-workspace",
		WorkspacesEnabled:  true,
		ParentRunID:        "test-parent-run-id",
		ExperimentID:       "test-exp",
		AuthType:           "kubernetes",
		Timeout:            "10s",
		InsecureSkipVerify: true,
		InjectUserEnvVars:  true,
	}

	handler, _ := NewMLflowTaskHandler(runtimeCfg)
	_, err := handler.RetrieveUserContainerEnvVars()

	require.Error(t, err)
	assert.Equal(t, "MLflow run ID is empty. Cannot inject MLFLOW_RUN_ID env var", err.Error())
}

func TestRetrieveUserContainerEnvVars_InjectVarsDisabled_Success(t *testing.T) {
	runtimeCfg := &commonmlflow.MLflowRuntimeConfig{
		Endpoint:           "http://localhost",
		Workspace:          "test-workspace",
		WorkspacesEnabled:  true,
		ParentRunID:        "test-parent-run-id",
		ExperimentID:       "test-exp",
		AuthType:           "kubernetes",
		Timeout:            "10s",
		InsecureSkipVerify: true,
		InjectUserEnvVars:  false,
	}

	handler, _ := NewMLflowTaskHandler(runtimeCfg)
	handler.nestedRunID = "test-run-id"
	envVars, err := handler.RetrieveUserContainerEnvVars()

	require.NoError(t, err)
	assert.Empty(t, envVars)
}

func TestRetrieveCustomProperties_WithRunID(t *testing.T) {
	runtimeCfg := &commonmlflow.MLflowRuntimeConfig{
		Endpoint:     "http://localhost",
		ParentRunID:  "test-parent-run-id",
		ExperimentID: "test-exp",
		AuthType:     "kubernetes",
		Timeout:      "10s",
	}

	handler, _ := NewMLflowTaskHandler(runtimeCfg)
	props := handler.GenerateCustomProperties(taskStartResult)

	assert.Equal(t, map[string]string{"plugins.mlflow.run_id": "test-run-id"}, props)
}

func TestRetrieveCustomProperties_EmptyRunID(t *testing.T) {
	runtimeCfg := &commonmlflow.MLflowRuntimeConfig{
		Endpoint:     "http://localhost",
		ParentRunID:  "test-parent-run-id",
		ExperimentID: "test-exp",
		AuthType:     "kubernetes",
		Timeout:      "10s",
	}

	handler, _ := NewMLflowTaskHandler(runtimeCfg)
	props := handler.GenerateCustomProperties(emptyTaskStartResult)

	assert.Nil(t, props)
}

func TestRetrieveCustomProperties_NilResult(t *testing.T) {
	runtimeCfg := &commonmlflow.MLflowRuntimeConfig{
		Endpoint:     "http://localhost",
		ParentRunID:  "test-parent-run-id",
		ExperimentID: "test-exp",
		AuthType:     "kubernetes",
		Timeout:      "10s",
	}

	handler, _ := NewMLflowTaskHandler(runtimeCfg)
	props := handler.GenerateCustomProperties(nil)

	assert.Nil(t, props)
}

func TestMapToParams_MapValue(t *testing.T) {
	params := map[string]interface{}{
		"nested_map": map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		},
	}

	result := mapToParams(params)

	require.Len(t, result, 1)
	assert.Equal(t, "nested_map", result[0].Key)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result[0].Value), &parsed))
	assert.Equal(t, "value1", parsed["key1"])
	assert.Equal(t, float64(42), parsed["key2"])
}

func TestMapToParams_ListValue(t *testing.T) {
	params := map[string]interface{}{
		"tags": []interface{}{"tag1", "tag2", "tag3"},
	}

	result := mapToParams(params)

	require.Len(t, result, 1)
	assert.Equal(t, "tags", result[0].Key)

	var parsed []interface{}
	require.NoError(t, json.Unmarshal([]byte(result[0].Value), &parsed))
	assert.Equal(t, []interface{}{"tag1", "tag2", "tag3"}, parsed)
}

func TestMapToParams_MixedValues(t *testing.T) {
	params := map[string]interface{}{
		"string_param": "hello",
		"int_param":    42,
		"float_param":  3.14,
		"bool_param":   true,
		"list_param":   []interface{}{1, "two", 3.0},
		"map_param":    map[string]interface{}{"nested": true},
	}

	result := mapToParams(params)
	require.Len(t, result, len(params))

	resultMap := make(map[string]string)
	for _, p := range result {
		resultMap[p.Key] = p.Value
	}

	assert.Equal(t, `"hello"`, resultMap["string_param"])
	assert.Equal(t, "42", resultMap["int_param"])
	assert.Equal(t, "3.14", resultMap["float_param"])
	assert.Equal(t, "true", resultMap["bool_param"])

	var listParsed []interface{}
	require.NoError(t, json.Unmarshal([]byte(resultMap["list_param"]), &listParsed))
	assert.Len(t, listParsed, 3)

	var mapParsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(resultMap["map_param"]), &mapParsed))
	assert.Equal(t, true, mapParsed["nested"])
}

func TestMapToParams_NestedMapWithList(t *testing.T) {
	params := map[string]interface{}{
		"config": map[string]interface{}{
			"layers":        []interface{}{64, 128, 256},
			"learning_rate": 0.001,
			"optimizer":     "adam",
		},
	}

	result := mapToParams(params)

	require.Len(t, result, 1)
	assert.Equal(t, "config", result[0].Key)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result[0].Value), &parsed))
	assert.Equal(t, "adam", parsed["optimizer"])
	assert.Equal(t, 0.001, parsed["learning_rate"])
	assert.Equal(t, []interface{}{float64(64), float64(128), float64(256)}, parsed["layers"])
}

func TestApplyCustomProperties_Success(t *testing.T) {
	runtimeCfg := &commonmlflow.MLflowRuntimeConfig{
		Endpoint:     "http://localhost",
		ParentRunID:  "test-parent-run-id",
		ExperimentID: "test-exp",
		AuthType:     "kubernetes",
		Timeout:      "10s",
	}

	handler, _ := NewMLflowTaskHandler(runtimeCfg)

	testProps := map[string]string{
		"plugins.mlflow.run_id": "custom-prop-run-id",
	}
	err := handler.ApplyCustomProperties(testProps)
	assert.NoError(t, err)
	assert.Equal(t, "custom-prop-run-id", handler.runtimeCfg.ParentRunID)
	assert.Equal(t, "custom-prop-run-id", handler.nestedRunID)
}

func testFormattedTags() []interface{} {
	return []interface{}{
		map[string]interface{}{
			"key":   "tag-key",
			"value": "tag-value",
		},
	}
}
