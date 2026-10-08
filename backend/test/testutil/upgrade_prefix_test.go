// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0.

package testutil_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	experimentparams "github.com/kubeflow/pipelines/backend/api/v2/go_http_client/experiment_client/experiment_service"
	"github.com/kubeflow/pipelines/backend/api/v2/go_http_client/experiment_model"
	pipelineparams "github.com/kubeflow/pipelines/backend/api/v2/go_http_client/pipeline_client/pipeline_service"
	uploadparams "github.com/kubeflow/pipelines/backend/api/v2/go_http_client/pipeline_upload_client/pipeline_upload_service"
	recurringparams "github.com/kubeflow/pipelines/backend/api/v2/go_http_client/recurring_run_client/recurring_run_service"
	"github.com/kubeflow/pipelines/backend/api/v2/go_http_client/recurring_run_model"
	runparams "github.com/kubeflow/pipelines/backend/api/v2/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/api/v2/go_http_client/run_model"
	clients "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v2"
	"github.com/kubeflow/pipelines/backend/test/config"
	"github.com/kubeflow/pipelines/backend/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpgradePreparationUsesOldServerAndVerificationUsesV2(t *testing.T) {
	oldURL, oldLegacy, oldCluster := *config.ApiUrl, *config.UseLegacyAPIPrefix, *config.InClusterRun
	oldTLS, oldSkipVerify := *config.TLSEnabled, *config.DisableTLSCheck
	t.Cleanup(func() {
		*config.ApiUrl, *config.UseLegacyAPIPrefix, *config.InClusterRun = oldURL, oldLegacy, oldCluster
		*config.TLSEnabled, *config.DisableTLSCheck = oldTLS, oldSkipVerify
	})
	var upgraded, seeded atomic.Bool
	paths := make(chan string, 20)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		prefix := "/apis/v2beta1"
		if upgraded.Load() {
			prefix = "/apis/v2"
		}
		w.Header().Set("Content-Type", "application/json")
		if !strings.HasPrefix(r.URL.Path, prefix+"/") {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"code":5,"message":"API prefix not served"}`))
			return
		}
		paths <- r.URL.Path
		switch strings.TrimPrefix(r.URL.Path, prefix) {
		case "/healthz":
			_, _ = w.Write([]byte(`{"multi_user":false,"pipeline_store":"database"}`))
		case "/experiments":
			assert.Equal(t, "Bearer token", r.Header.Get("Authorization"))
			var body map[string]any
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, "before-upgrade", body["display_name"])
			seeded.Store(true)
			_, _ = w.Write([]byte(`{"experiment_id":"seed","display_name":"before-upgrade"}`))
		case "/experiments/seed":
			assert.True(t, seeded.Load(), "verification must read preparation data")
			_, _ = w.Write([]byte(`{"experiment_id":"seed","display_name":"before-upgrade"}`))
		case "/pipelines/upload":
			assert.NoError(t, r.ParseMultipartForm(1<<20))
			if r.MultipartForm != nil {
				defer r.MultipartForm.RemoveAll()
				assert.NotEmpty(t, r.MultipartForm.File["uploadfile"])
			}
			_, _ = w.Write([]byte(`{"pipeline_id":"pipeline"}`))
		case "/pipelines":
			_, _ = w.Write([]byte(`{"pipelines":[{"pipeline_id":"pipeline"}]}`))
		case "/runs":
			_, _ = w.Write([]byte(`{"run_id":"run"}`))
		case "/recurringruns":
			_, _ = w.Write([]byte(`{"recurring_run_id":"recurring"}`))
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	*config.ApiUrl, *config.InClusterRun, *config.TLSEnabled, *config.DisableTLSCheck = server.URL, false, false, false
	*config.UseLegacyAPIPrefix = false
	require.Error(t, testutil.WaitForReady(time.Millisecond), "a missing stable endpoint must not count as ready")
	*config.UseLegacyAPIPrefix = true
	require.NoError(t, testutil.WaitForReady(time.Millisecond))

	experiment, err := clients.NewMultiUserExperimentClient(nil, "token", false, nil)
	require.NoError(t, err)
	created, err := experiment.Create(experimentparams.NewExperimentServiceCreateExperimentParams().WithExperiment(&experiment_model.V2Experiment{DisplayName: "before-upgrade"}))
	require.NoError(t, err)
	require.Equal(t, "seed", created.ExperimentID)
	pipeline, err := clients.NewPipelineClient(nil, false, nil)
	require.NoError(t, err)
	_, _, _, err = pipeline.List(pipelineparams.NewPipelineServiceListPipelinesParams())
	require.NoError(t, err)
	upload, err := clients.NewPipelineUploadClient(nil, false, nil)
	require.NoError(t, err)
	filename := filepath.Join(t.TempDir(), "pipeline.yaml")
	require.NoError(t, os.WriteFile(filename, []byte("pipelineInfo:\n  name: test\n"), 0o600))
	_, err = upload.UploadFile(filename, uploadparams.NewUploadPipelineParams())
	require.NoError(t, err)
	run, err := clients.NewRunClient(nil, false, nil)
	require.NoError(t, err)
	_, err = run.Create(runparams.NewRunServiceCreateRunParams().WithRun(&run_model.V2Run{DisplayName: "seed"}))
	require.NoError(t, err)
	recurring, err := clients.NewRecurringRunClient(nil, false, nil)
	require.NoError(t, err)
	_, err = recurring.Create(recurringparams.NewRecurringRunServiceCreateRecurringRunParams().WithRecurringRun(&recurring_run_model.V2RecurringRun{DisplayName: "seed"}))
	require.NoError(t, err)

	upgraded.Store(true)
	*config.UseLegacyAPIPrefix = false
	require.NoError(t, testutil.WaitForReady(time.Millisecond))
	verification, err := clients.NewExperimentClient(nil, false, nil)
	require.NoError(t, err)
	persisted, err := verification.Get(experimentparams.NewExperimentServiceGetExperimentParams().WithExperimentID(created.ExperimentID))
	require.NoError(t, err)
	require.Equal(t, created.ExperimentID, persisted.ExperimentID)
	require.Equal(t, created.DisplayName, persisted.DisplayName)
	close(paths)
	var received []string
	for path := range paths {
		received = append(received, path)
	}
	require.Equal(t, []string{
		"/apis/v2beta1/healthz", "/apis/v2beta1/experiments", "/apis/v2beta1/pipelines",
		"/apis/v2beta1/pipelines/upload", "/apis/v2beta1/runs", "/apis/v2beta1/recurringruns",
		"/apis/v2/healthz", "/apis/v2/experiments/seed",
	}, received)
}
