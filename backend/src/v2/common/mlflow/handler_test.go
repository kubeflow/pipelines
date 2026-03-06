package mlflow

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/mlflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSAToken(t *testing.T) func() {
	t.Helper()
	setupFakeKubernetesConfig(t, "test-sa-token")
	return func() {} // cleanup handled by t.Cleanup in setupFakeKubernetesConfig
}

func testPluginConfig(endpoint string) commonmlflow.PluginConfig {
	workspacesEnabled := false
	experimentDesc := "test-description"
	injectUserEnvVars := false

	return commonmlflow.PluginConfig{
		Endpoint: endpoint,
		Timeout:  "10s",
		Settings: &commonmlflow.MLflowPluginSettings{
			WorkspacesEnabled:     &workspacesEnabled,
			ExperimentDescription: &experimentDesc,
			DefaultExperimentName: "test-exp",
			KFPBaseURL:            "test-url",
			InjectUserEnvVars:     &injectUserEnvVars,
		},
	}
}

func testTaskInfo_TaskStart(parentRunID string) TaskInfo {
	return TaskInfo{
		ParentRunID:  parentRunID,
		ExperimentID: "exp-1",
		AuthType:     "kubernetes",
	}
}

func testTaskInfo_TaskEnd(runID string) TaskInfo {
	return TaskInfo{
		RunID:        runID,
		ExperimentID: "exp-1",
		AuthType:     "kubernetes",
		RunEndTime:   10,
	}
}

func testMetrics() map[string]float64 {
	return map[string]float64{
		"test-metric": 0.5,
	}
}

func testParams() map[string]string {
	return map[string]string{
		"test-param": "test-value",
	}
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

func TestOnTaskStart_NilPluginConfig_ReturnsEmptyString(t *testing.T) {
	handler := NewHandler()
	nestedRunID, err := handler.OnTaskStart(context.Background(), "test-task", testTaskInfo_TaskStart("r1"), nil)

	assert.Empty(t, nestedRunID)
	require.NoError(t, err)
}

func TestOnTaskStart_MissingRunID_Failure(t *testing.T) {
	handler := NewHandler()
	nestedRunID, err := handler.OnTaskStart(context.Background(), "test-task", TaskInfo{ExperimentID: "1", AuthType: "kubernetes"}, testPluginConfig("http://localhost"))
	assert.Empty(t, nestedRunID)

	require.Error(t, err)
	assert.Equal(t, "ParentRunID and ExperimentID are both required to create nested MLflow run", err.Error())
}

func TestOnTaskStart_MissingExperimentID_Failure(t *testing.T) {
	handler := NewHandler()
	nestedRunID, err := handler.OnTaskStart(context.Background(), "test-task", TaskInfo{RunID: "r1", AuthType: "kubernetes"}, testPluginConfig("http://localhost"))
	assert.Empty(t, nestedRunID)

	require.Error(t, err)
	assert.Equal(t, "ParentRunID and ExperimentID are both required to create nested MLflow run", err.Error())
}

func TestOnTaskStart_Success(t *testing.T) {
	cleanup := setupSAToken(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	}))
	defer server.Close()

	handler := NewHandler()

	taskInfo := testTaskInfo_TaskStart("r1")
	runID, err := handler.OnTaskStart(context.Background(), "test-task", taskInfo, testPluginConfig(server.URL))
	require.NoError(t, err)
	require.NotEmpty(t, runID)

}

func TestOnTaskStart_MLflowFailure_ReturnsEmptyNestedRunID(t *testing.T) {
	cleanup := setupSAToken(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error_code":"INTERNAL_ERROR","message":"server down"}`))
	}))
	defer server.Close()

	handler := NewHandler()
	nestedRunID, err := handler.OnTaskStart(context.Background(), "test-task", testTaskInfo_TaskStart("r1"), testPluginConfig(server.URL))
	require.Error(t, err)
	require.Empty(t, nestedRunID)
}

func TestOnTaskEnd_MissingRunID_Failure(t *testing.T) {
	handler := NewHandler()
	err := handler.OnTaskEnd(context.Background(), TaskInfo{ExperimentID: "test-experiment", AuthType: "kubernetes"}, testMetrics(), testParams(), testPluginConfig("test-endpoint"))

	require.Error(t, err)
	assert.Equal(t, "RunID and ExperimentID are both required to update nested MLflow run", err.Error())
}

func TestOnTaskEnd_MissingExperimentID_Failure(t *testing.T) {
	handler := NewHandler()
	err := handler.OnTaskEnd(context.Background(), TaskInfo{RunID: "test-run", AuthType: "kubernetes"}, testMetrics(), testParams(), testPluginConfig("test-endpoint"))

	require.Error(t, err)
	assert.Equal(t, "RunID and ExperimentID are both required to update nested MLflow run", err.Error())
}

func TestOnTaskEnd_EmptyMetrics_Success(t *testing.T) {
	cleanup := setupSAToken(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	}))
	defer server.Close()

	handler := NewHandler()
	err := handler.OnTaskEnd(context.Background(), testTaskInfo_TaskEnd("run-1"), map[string]float64{}, testParams(), testPluginConfig(server.URL))
	require.NoError(t, err)
}

func TestOnTaskEnd_EmptyParams_Success(t *testing.T) {
	cleanup := setupSAToken(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	}))
	defer server.Close()

	handler := NewHandler()
	err := handler.OnTaskEnd(context.Background(), testTaskInfo_TaskEnd("run-1"), testMetrics(), map[string]string{}, testPluginConfig(server.URL))
	require.NoError(t, err)
}

func TestOnTaskEnd_Success(t *testing.T) {
	cleanup := setupSAToken(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	}))
	defer server.Close()

	handler := NewHandler()

	err := handler.OnTaskEnd(context.Background(), testTaskInfo_TaskEnd("r1"), testMetrics(), testParams(), testPluginConfig(server.URL))
	require.NoError(t, err)
}

func TestOnTaskEnd_EmptyRunID(t *testing.T) {
	cleanup := setupSAToken(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	}))
	defer server.Close()

	handler := NewHandler()

	err := handler.OnTaskEnd(context.Background(), testTaskInfo_TaskEnd(""), testMetrics(), testParams(), testPluginConfig(server.URL))
	require.Error(t, err)
	assert.Equal(t, "RunID and ExperimentID are both required to update nested MLflow run", err.Error())
}
