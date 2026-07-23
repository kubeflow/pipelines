package mlflow

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	apiserverPlugins "github.com/kubeflow/pipelines/backend/src/apiserver/plugins"
	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/plugins/mlflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestEnsureExperimentExists_NilRequestContext(t *testing.T) {
	_, err := EnsureExperimentExists(context.Background(), nil, "", "exp", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "MLflow request context is required")
}

func TestEnsureExperimentExists_ByID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/2.0/mlflow/experiments/get", r.URL.Path)
		assert.Equal(t, "42", r.URL.Query().Get("experiment_id"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"experiment":{"experiment_id":"42","name":"found-exp"}}`))
	}))
	defer server.Close()

	mlflowCtx := newTestMLflowRequestContext(t, server.URL)
	exp, err := EnsureExperimentExists(context.Background(), mlflowCtx, "42", "", nil)
	require.NoError(t, err)
	require.NotNil(t, exp)
	assert.Equal(t, "42", exp.ID)
	assert.Equal(t, "found-exp", exp.Name)
}

func TestEnsureExperimentExists_ByIDNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error_code":"RESOURCE_DOES_NOT_EXIST"}`))
	}))
	defer server.Close()

	mlflowCtx := newTestMLflowRequestContext(t, server.URL)
	_, err := EnsureExperimentExists(context.Background(), mlflowCtx, "999", "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "experiment ID")
	assert.Contains(t, err.Error(), "not found")
}

func TestEnsureExperimentExists_ByName_Existing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/2.0/mlflow/experiments/get-by-name", r.URL.Path)
		assert.Equal(t, "existing-exp", r.URL.Query().Get("experiment_name"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"experiment":{"experiment_id":"50","name":"existing-exp"}}`))
	}))
	defer server.Close()

	mlflowCtx := newTestMLflowRequestContext(t, server.URL)
	exp, err := EnsureExperimentExists(context.Background(), mlflowCtx, "", "existing-exp", nil)
	require.NoError(t, err)
	require.NotNil(t, exp)
	assert.Equal(t, "50", exp.ID)
	assert.Equal(t, "existing-exp", exp.Name)
}

func TestEnsureExperimentExists_ByName_NotFound_Create(t *testing.T) {
	var callCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/2.0/mlflow/experiments/get-by-name":
			callCount++
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error_code":"RESOURCE_DOES_NOT_EXIST"}`))
		case "/api/2.0/mlflow/experiments/create":
			callCount++
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"experiment_id":"60"}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	mlflowCtx := newTestMLflowRequestContext(t, server.URL)
	exp, err := EnsureExperimentExists(context.Background(), mlflowCtx, "", "new-exp", nil)
	require.NoError(t, err)
	require.NotNil(t, exp)
	assert.Equal(t, "60", exp.ID)
	assert.Equal(t, "new-exp", exp.Name)
	assert.Equal(t, 2, callCount) // get-by-name + create
}

func TestEnsureExperimentExists_ByName_GetByNameError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/2.0/mlflow/experiments/get-by-name", r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error_code":"INTERNAL_ERROR"}`))
	}))
	defer server.Close()

	mlflowCtx := newTestMLflowRequestContext(t, server.URL)
	_, err := EnsureExperimentExists(context.Background(), mlflowCtx, "", "test-exp", nil)
	require.Error(t, err)
	// Should not contain "not found" since it's a different error
	assert.NotContains(t, err.Error(), "not found")
}

func TestCreateExperiment_AlreadyExistsRace(t *testing.T) {
	var callCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/2.0/mlflow/experiments/create":
			callCount++
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error_code":"RESOURCE_ALREADY_EXISTS"}`))
		case "/api/2.0/mlflow/experiments/get-by-name":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"experiment":{"experiment_id":"100","name":"race-exp"}}`))
		}
	}))
	defer server.Close()

	mlflowCtx := newTestMLflowRequestContext(t, server.URL)
	exp, err := CreateExperiment(context.Background(), mlflowCtx, "race-exp", nil)
	require.NoError(t, err)
	require.NotNil(t, exp)
	assert.Equal(t, "100", exp.ID)
	assert.Equal(t, "race-exp", exp.Name)
	assert.Equal(t, 1, callCount)
}

func TestCreateExperiment_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/2.0/mlflow/experiments/create", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"experiment_id":"123"}`))
	}))
	defer server.Close()

	mlflowCtx := newTestMLflowRequestContext(t, server.URL)
	exp, err := CreateExperiment(context.Background(), mlflowCtx, "new-experiment", nil)
	require.NoError(t, err)
	require.NotNil(t, exp)
	assert.Equal(t, "123", exp.ID)
	assert.Equal(t, "new-experiment", exp.Name)
}

func TestCreateExperiment_SuccessWithDescription(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/2.0/mlflow/experiments/create", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		_ = json.Unmarshal(body, &req)
		assert.Equal(t, "test description", req["description"])
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"experiment_id":"456"}`))
	}))
	defer server.Close()

	description := "test description"
	mlflowCtx := newTestMLflowRequestContext(t, server.URL)
	exp, err := CreateExperiment(context.Background(), mlflowCtx, "described-exp", &description)
	require.NoError(t, err)
	require.NotNil(t, exp)
	assert.Equal(t, "456", exp.ID)
	assert.Equal(t, "described-exp", exp.Name)
}

func TestCreateExperiment_NonRaceError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/2.0/mlflow/experiments/create", r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error_code":"INTERNAL_ERROR","message":"database unavailable"}`))
	}))
	defer server.Close()

	mlflowCtx := newTestMLflowRequestContext(t, server.URL)
	_, err := CreateExperiment(context.Background(), mlflowCtx, "failing-exp", nil)
	require.Error(t, err)
}

func TestBuildRunURL_EmptyExperimentID(t *testing.T) {
	requestCtx := newTestMLflowRequestContext(t, "https://mlflow.example.com")
	url := BuildRunURL(requestCtx, "", "run-1", nil)
	assert.Empty(t, url)
}

func TestBuildRunURL_EmptyRunID(t *testing.T) {
	requestCtx := newTestMLflowRequestContext(t, "https://mlflow.example.com")
	url := BuildRunURL(requestCtx, "exp-1", "", nil)
	assert.Empty(t, url)
}

func TestBuildRunURL_NoBaseURL(t *testing.T) {
	requestCtx := &commonmlflow.RequestContext{
		BaseURL:           nil,
		WorkspacesEnabled: false,
	}
	url := BuildRunURL(requestCtx, "exp-1", "run-1", nil)
	assert.Empty(t, url)
}

func TestBuildRunURL_Basic(t *testing.T) {
	requestCtx := newTestMLflowRequestContext(t, "https://mlflow.example.com")
	requestCtx.WorkspacesEnabled = false
	url := BuildRunURL(requestCtx, "exp-1", "run-1", nil)
	assert.Equal(t, "https://mlflow.example.com/#/experiments/exp-1/runs/run-1", url)
}

func TestBuildRunURL_WithWorkspace(t *testing.T) {
	requestCtx := newTestMLflowRequestContext(t, "https://mlflow.example.com")
	requestCtx.WorkspacesEnabled = true
	requestCtx.Workspace = "team-a"
	url := BuildRunURL(requestCtx, "exp-2", "run-2", nil)
	assert.Equal(t, "https://mlflow.example.com/#/experiments/exp-2/runs/run-2?workspace=team-a", url)
}

func TestBuildRunURL_WithMLflowBaseURL(t *testing.T) {
	requestCtx := newTestMLflowRequestContext(t, "https://mlflow-api.example.com")
	requestCtx.WorkspacesEnabled = false
	settings := &commonmlflow.MLflowPluginSettings{
		MLflowBaseURL: "https://mlflow-ui.example.com",
	}
	url := BuildRunURL(requestCtx, "exp-1", "run-1", settings)
	assert.Equal(t, "https://mlflow-ui.example.com/#/experiments/exp-1/runs/run-1", url)
}

func TestBuildRunURL_WithUIPathPrefix(t *testing.T) {
	requestCtx := newTestMLflowRequestContext(t, "https://app.example.com")
	requestCtx.WorkspacesEnabled = false
	settings := &commonmlflow.MLflowPluginSettings{
		MLflowUIPathPrefix: "/mlflow-ui",
	}
	url := BuildRunURL(requestCtx, "exp-1", "run-1", settings)
	assert.Equal(t, "https://app.example.com/mlflow-ui/#/experiments/exp-1/runs/run-1", url)
}

func TestBuildRunURL_WithUIPathPrefix_NoLeadingSlash(t *testing.T) {
	requestCtx := newTestMLflowRequestContext(t, "https://app.example.com")
	requestCtx.WorkspacesEnabled = false
	settings := &commonmlflow.MLflowPluginSettings{
		MLflowUIPathPrefix: "mlflow",
	}
	url := BuildRunURL(requestCtx, "exp-1", "run-1", settings)
	assert.Equal(t, "https://app.example.com/mlflow/#/experiments/exp-1/runs/run-1", url)
}

func TestBuildRunURL_URLEscaping(t *testing.T) {
	requestCtx := newTestMLflowRequestContext(t, "https://mlflow.example.com")
	requestCtx.WorkspacesEnabled = true
	requestCtx.Workspace = "Team A"
	url := BuildRunURL(requestCtx, "exp/1", "run/1", nil)
	assert.Equal(t, "https://mlflow.example.com/#/experiments/exp%2F1/runs/run%2F1?workspace=team+a", url)
}

func TestSuccessfulPluginOutput(t *testing.T) {
	output := SuccessfulPluginOutput("exp-1", "my-exp", "run-1", "https://mlflow.example.com/runs/run-1")
	require.NotNil(t, output)
	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_SUCCEEDED, output.State)
	assert.Empty(t, output.StateMessage)
	require.Contains(t, output.Entries, "experiment_id")
	require.Contains(t, output.Entries, "experiment_name")
	require.Contains(t, output.Entries, "root_run_id")
	require.Contains(t, output.Entries, "run_url")
	assert.Equal(t, "exp-1", output.Entries["experiment_id"].Value.GetStringValue())
	assert.Equal(t, "my-exp", output.Entries["experiment_name"].Value.GetStringValue())
	assert.Equal(t, "run-1", output.Entries["root_run_id"].Value.GetStringValue())
	assert.Equal(t, "https://mlflow.example.com/runs/run-1", output.Entries["run_url"].Value.GetStringValue())
	assert.Equal(t, apiv2beta1.MetadataValue_URL, output.Entries["run_url"].GetRenderType())
}

func TestSuccessfulPluginOutput_EmptyValues(t *testing.T) {
	output := SuccessfulPluginOutput("", "", "", "")
	require.NotNil(t, output)
	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_SUCCEEDED, output.State)
	assert.Empty(t, output.StateMessage)
	// Empty values should not create entries
	assert.Empty(t, output.Entries)
}

func TestFailedPluginOutput(t *testing.T) {
	output := FailedPluginOutput("exp-1", "my-exp", "run-1", "https://mlflow.example.com/runs/run-1", "connection timeout")
	require.NotNil(t, output)
	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_FAILED, output.State)
	assert.Equal(t, "connection timeout", output.StateMessage)
	require.Contains(t, output.Entries, "experiment_id")
	require.Contains(t, output.Entries, "experiment_name")
	require.Contains(t, output.Entries, "root_run_id")
	require.Contains(t, output.Entries, "run_url")
	assert.Equal(t, "exp-1", output.Entries["experiment_id"].Value.GetStringValue())
	assert.Equal(t, "my-exp", output.Entries["experiment_name"].Value.GetStringValue())
	assert.Equal(t, "run-1", output.Entries["root_run_id"].Value.GetStringValue())
	assert.Equal(t, "https://mlflow.example.com/runs/run-1", output.Entries["run_url"].Value.GetStringValue())
	assert.Equal(t, apiv2beta1.MetadataValue_URL, output.Entries["run_url"].GetRenderType())
}

func TestFailedPluginOutput_EmptyValues(t *testing.T) {
	output := FailedPluginOutput("", "", "", "", "error message")
	require.NotNil(t, output)
	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_FAILED, output.State)
	assert.Equal(t, "error message", output.StateMessage)
	// Empty values should not create entries
	assert.Empty(t, output.Entries)
}

func TestSyncParentAndNestedRuns_NilRequestContext(t *testing.T) {
	errors := SyncParentAndNestedRuns(context.Background(), nil, "parent-run", "exp-1", apiserverPlugins.RunSyncModeTerminal, "FINISHED", nil)
	require.Len(t, errors, 1)
	assert.Contains(t, errors[0], "MLflow request context is required")
}

func TestSyncParentAndNestedRuns_EmptyParentRunID(t *testing.T) {
	requestCtx := newTestMLflowRequestContext(t, "https://mlflow.example.com")
	errors := SyncParentAndNestedRuns(context.Background(), requestCtx, "", "exp-1", apiserverPlugins.RunSyncModeTerminal, "FINISHED", nil)
	require.Len(t, errors, 1)
	assert.Contains(t, errors[0], "MLflow parent run_id is required")
}

func TestSyncParentAndNestedRuns_UnsupportedMode(t *testing.T) {
	requestCtx := newTestMLflowRequestContext(t, "https://mlflow.example.com")
	errors := SyncParentAndNestedRuns(context.Background(), requestCtx, "parent-run", "exp-1", "invalid-mode", "FINISHED", nil)
	require.Len(t, errors, 1)
	assert.Contains(t, errors[0], "unsupported MLflow run sync mode")
}

func TestSyncParentAndNestedRuns_TerminalMode(t *testing.T) {
	var updateCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/2.0/mlflow/runs/update":
			body, _ := io.ReadAll(r.Body)
			var req map[string]interface{}
			_ = json.Unmarshal(body, &req)
			updateCalls = append(updateCalls, fmt.Sprintf("%s:%s", req["run_id"], req["status"]))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		case "/api/2.0/mlflow/runs/search":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"runs":[]}`))
		}
	}))
	defer server.Close()

	requestCtx := newTestMLflowRequestContext(t, server.URL)
	endTime := int64(1234567890)
	errors := SyncParentAndNestedRuns(context.Background(), requestCtx, "parent-1", "exp-1", apiserverPlugins.RunSyncModeTerminal, "FINISHED", &endTime)
	assert.Empty(t, errors)
	require.Len(t, updateCalls, 1)
	assert.Equal(t, "parent-1:FINISHED", updateCalls[0])
}

func TestSyncParentAndNestedRuns_RetryMode(t *testing.T) {
	var updateCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/2.0/mlflow/runs/update":
			body, _ := io.ReadAll(r.Body)
			var req map[string]interface{}
			_ = json.Unmarshal(body, &req)
			updateCalls = append(updateCalls, fmt.Sprintf("%s:%s", req["run_id"], req["status"]))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		case "/api/2.0/mlflow/runs/search":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"runs":[]}`))
		}
	}))
	defer server.Close()

	requestCtx := newTestMLflowRequestContext(t, server.URL)
	errors := SyncParentAndNestedRuns(context.Background(), requestCtx, "parent-2", "exp-1", apiserverPlugins.RunSyncModeRetry, "FAILED", nil)
	assert.Empty(t, errors)
	require.Len(t, updateCalls, 1)
	assert.Equal(t, "parent-2:RUNNING", updateCalls[0])
}

func TestSyncParentAndNestedRuns_WithNestedRuns(t *testing.T) {
	var updateCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/2.0/mlflow/runs/update":
			body, _ := io.ReadAll(r.Body)
			var req map[string]interface{}
			_ = json.Unmarshal(body, &req)
			updateCalls = append(updateCalls, fmt.Sprintf("%s:%s", req["run_id"], req["status"]))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		case "/api/2.0/mlflow/runs/search":
			body, _ := io.ReadAll(r.Body)
			var req map[string]interface{}
			_ = json.Unmarshal(body, &req)
			filter := req["filter"].(string)
			if strings.Contains(filter, "parent-1") {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"runs":[{"info":{"run_id":"nested-1","status":"RUNNING"}}]}`))
			} else {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"runs":[]}`))
			}
		}
	}))
	defer server.Close()

	requestCtx := newTestMLflowRequestContext(t, server.URL)
	errors := SyncParentAndNestedRuns(context.Background(), requestCtx, "parent-1", "exp-1", apiserverPlugins.RunSyncModeTerminal, "FAILED", nil)
	assert.Empty(t, errors)
	require.Len(t, updateCalls, 2)
	assert.Contains(t, updateCalls, "parent-1:FAILED")
	assert.Contains(t, updateCalls, "nested-1:FAILED")
}

func TestShouldSyncNestedRun_TerminalMode(t *testing.T) {
	tests := []struct {
		status string
		want   bool
	}{
		{"RUNNING", true},
		{"SCHEDULED", true},
		{"FINISHED", false},
		{"FAILED", false},
		{"KILLED", false},
		{"finished", false},
		{"failed", false},
		{"killed", false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := shouldSyncNestedRun(apiserverPlugins.RunSyncModeTerminal, tt.status)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestShouldSyncNestedRun_RetryMode(t *testing.T) {
	tests := []struct {
		status string
		want   bool
	}{
		{"FAILED", true},
		{"KILLED", true},
		{"failed", true},
		{"killed", true},
		{"RUNNING", false},
		{"FINISHED", false},
		{"SCHEDULED", false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := shouldSyncNestedRun(apiserverPlugins.RunSyncModeRetry, tt.status)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestShouldSyncNestedRun_UnsupportedMode(t *testing.T) {
	got := shouldSyncNestedRun("unsupported", "RUNNING")
	assert.False(t, got)
}

func TestCreateRunWithKFPTags(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer bearer-secret", r.Header.Get("Authorization"))
		assert.Equal(t, "ns1", r.Header.Get("X-MLflow-Workspace"))
		switch r.URL.Path {
		case "/api/2.0/mlflow/runs/create":
			body, _ := io.ReadAll(r.Body)
			require.NoError(t, json.Unmarshal(body, &receivedBody))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"run":{"info":{"run_id":"mlflow-parent-run-1"}}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	mlflowCtx := newTestMLflowRequestContext(t, server.URL)

	run := &apiserverPlugins.PendingRun{
		RunID:             "kfp-run-1",
		PipelineID:        "pipeline-1",
		PipelineVersionID: "pipeline-version-1",
	}
	tags := BuildKFPTags(run, "", "")
	mlflowRunID, err := mlflowCtx.Client.CreateRun(context.Background(), "exp-1", "sample-run", tags)
	require.NoError(t, err)
	assert.Equal(t, "mlflow-parent-run-1", mlflowRunID)

	// Verify tags were included in the CreateRun request body. The four KFP
	// tags plus the idempotency-key tag used for CreateRun retries.
	rawTags, ok := receivedBody["tags"].([]interface{})
	require.True(t, ok, "tags should be present in CreateRun body")
	assert.Len(t, rawTags, 5)
}

func TestBuildKFPTags(t *testing.T) {
	run := &apiserverPlugins.PendingRun{
		RunID:             "kfp-run-1",
		Namespace:         "ns-1",
		PipelineID:        "pipeline-1",
		PipelineVersionID: "pipeline-version-1",
	}
	tags := BuildKFPTags(run, "", "")
	require.Len(t, tags, 5)
	assert.Contains(t, tags, commonmlflow.Tag{Key: TagKFPRunID, Value: "kfp-run-1"})
	assert.Contains(t, tags, commonmlflow.Tag{Key: commonmlflow.IdempotencyTagKey, Value: "kfp-run-1"})
	assert.Contains(t, tags, commonmlflow.Tag{Key: TagKFPRunURL, Value: ""})
	assert.Contains(t, tags, commonmlflow.Tag{Key: TagKFPPipelineID, Value: "pipeline-1"})
	assert.Contains(t, tags, commonmlflow.Tag{Key: TagKFPPipelineVersionID, Value: "pipeline-version-1"})
}

func TestBuildKFPTags_WithBaseURL(t *testing.T) {
	run := &apiserverPlugins.PendingRun{
		RunID:     "run-1",
		Namespace: "ns-1",
	}
	tags := BuildKFPTags(run, "https://kfp.example.com", "")
	require.Len(t, tags, 3)
	assert.Contains(t, tags, commonmlflow.Tag{
		Key:   TagKFPRunURL,
		Value: "https://kfp.example.com/#/runs/details/run-1",
	})
}

func TestBuildKFPTags_WithCustomPathTemplate(t *testing.T) {
	run := &apiserverPlugins.PendingRun{
		RunID:     "run-b",
		Namespace: "proj-1",
	}
	tmpl := "/demo/console/pipelines/{namespace}/runs/{run_id}"
	tags := BuildKFPTags(run, "https://console.example.com", tmpl)
	require.Len(t, tags, 3)
	assert.Contains(t, tags, commonmlflow.Tag{
		Key:   TagKFPRunURL,
		Value: "https://console.example.com/demo/console/pipelines/proj-1/runs/run-b",
	})
}

func TestBuildKFPTags_NilRun(t *testing.T) {
	assert.Nil(t, BuildKFPTags(nil, "", ""))
}

func TestEnsureExperimentExists(t *testing.T) {
	t.Run("returns existing experiment from get-by-name", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer bearer-secret", r.Header.Get("Authorization"))
			assert.Equal(t, "ns1", r.Header.Get("X-MLflow-Workspace"))
			assert.Equal(t, "/api/2.0/mlflow/experiments/get-by-name", r.URL.Path)
			assert.Equal(t, "my-exp", r.URL.Query().Get("experiment_name"))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"experiment":{"experiment_id":"42","name":"my-exp"}}`))
		}))
		defer server.Close()

		mlflowCtx := newTestMLflowRequestContext(t, server.URL)
		defaultDesc := DefaultExperimentDescription
		exp, err := EnsureExperimentExists(context.Background(), mlflowCtx, "", "my-exp", &defaultDesc)
		require.NoError(t, err)
		require.NotNil(t, exp)
		assert.Equal(t, "42", exp.ID)
		assert.Equal(t, "my-exp", exp.Name)
	})

	t.Run("creates experiment when get-by-name returns not found", func(t *testing.T) {
		var callCount int
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer bearer-secret", r.Header.Get("Authorization"))
			assert.Equal(t, "ns1", r.Header.Get("X-MLflow-Workspace"))
			switch r.URL.Path {
			case "/api/2.0/mlflow/experiments/get-by-name":
				callCount++
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error_code":"RESOURCE_DOES_NOT_EXIST","message":"not found"}`))
			case "/api/2.0/mlflow/experiments/create":
				bodyBytes, _ := io.ReadAll(r.Body)
				assert.Contains(t, string(bodyBytes), `"name":"my-exp"`)
				assert.Contains(t, string(bodyBytes), `"description":"Created by Kubeflow Pipelines"`)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"experiment_id":"99"}`))
			default:
				t.Fatalf("unexpected path %s", r.URL.Path)
			}
		}))
		defer server.Close()

		defaultDesc := DefaultExperimentDescription
		mlflowCtx := newTestMLflowRequestContext(t, server.URL)
		exp, err := EnsureExperimentExists(context.Background(), mlflowCtx, "", "my-exp", &defaultDesc)
		require.NoError(t, err)
		require.NotNil(t, exp)
		assert.Equal(t, "99", exp.ID)
		assert.Equal(t, "my-exp", exp.Name)
		assert.Equal(t, 1, callCount)
	})
}

func TestBuildPluginOutput(t *testing.T) {
	output := SuccessfulPluginOutput("exp-1", "my-exp", "run-1", "https://mlflow.example/runs/run-1")
	require.NotNil(t, output)
	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_SUCCEEDED, output.State)
	require.Contains(t, output.Entries, "run_url")
	require.NotNil(t, output.Entries["run_url"].RenderType)
	assert.Equal(t, apiv2beta1.MetadataValue_URL, *output.Entries["run_url"].RenderType)
}

func TestSetPendingRunPluginOutput(t *testing.T) {
	// Start with a PendingRun that already has another plugin's output.
	existing := `{"other":{"state":"PLUGIN_SUCCEEDED"}}`
	run := &apiserverPlugins.PendingRun{
		RunID:         "run-1",
		PluginsOutput: &existing,
	}
	mlflowOutput := SuccessfulPluginOutput("exp-1", "my-exp", "run-1", "https://mlflow.example/runs/run-1")
	err := apiserverPlugins.SetPendingRunPluginOutput(run, "mlflow", mlflowOutput)
	require.NoError(t, err)
	require.NotNil(t, run.PluginsOutput)

	var envelope apiserverPlugins.PluginsOutputEnvelope
	require.NoError(t, json.Unmarshal([]byte(*run.PluginsOutput), &envelope))
	assert.NotNil(t, envelope.Plugins["mlflow"], "mlflow entry should be set")
	assert.NotNil(t, envelope.Plugins["other"], "pre-existing 'other' entry should be preserved")
	var parsed apiv2beta1.PluginOutput
	require.NoError(t, protojson.Unmarshal(envelope.Plugins["mlflow"], &parsed))
	assert.Equal(t, apiv2beta1.PluginState_PLUGIN_SUCCEEDED, parsed.State)
	assert.Contains(t, parsed.Entries, "experiment_id")
}
