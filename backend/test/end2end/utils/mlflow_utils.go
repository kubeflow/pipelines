// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Package utils provides helpers shared across end-to-end tests.
package utils

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	runparams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
	mlflowclient "github.com/kubeflow/pipelines/backend/src/common/mlflow"
	apiserver "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v2"
	"github.com/kubeflow/pipelines/backend/test/config"
	"github.com/kubeflow/pipelines/backend/test/logger"
	apitests "github.com/kubeflow/pipelines/backend/test/v2/api"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

const (
	mlflowEndpointEnv    = "MLFLOW_TRACKING_URI"
	mlflowInsecureTLSEnv = "MLFLOW_TRACKING_INSECURE_TLS"
	mlflowBearerTokenEnv = "MLFLOW_BEARER_TOKEN"
	mlflowWorkspaceEnv   = "MLFLOW_WORKSPACE"
	mlflowPluginKey      = "mlflow"
)

func getMLflowClient(endpoint string) (*mlflowclient.Client, error) {
	insecure := strings.EqualFold(os.Getenv(mlflowInsecureTLSEnv), "true")
	workspace := os.Getenv(mlflowWorkspaceEnv)
	bearerToken := os.Getenv(mlflowBearerTokenEnv)
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: insecure, //nolint:gosec
			},
		},
	}
	client, err := mlflowclient.NewClient(mlflowclient.Config{
		Endpoint:          endpoint,
		HTTPClient:        httpClient,
		BearerToken:       bearerToken,
		WorkspacesEnabled: workspace != "",
		Workspace:         workspace,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MLflow client: %w", err)
	}
	if bearerToken != "" {
		logger.Log("MLflow client initialized with bearer token auth")
	}
	if workspace != "" {
		logger.Log("MLflow client initialized with workspace header: %s", workspace)
	}
	if insecure {
		logger.Log("MLflow client initialized with InsecureSkipVerify=true")
	}
	return client, nil
}

// RetryPipelineRun retries a failed/terminated KFP pipeline run.
func RetryPipelineRun(runClient *apiserver.RunClient, runID string) {
	ginkgo.GinkgoHelper()
	retryParams := runparams.NewRunServiceRetryRunParams()
	retryParams.RunID = runID
	err := runClient.Retry(retryParams)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(),
		fmt.Sprintf("Failed to retry run %s", runID))
	logger.Log("Retried Pipeline Run, runId=%s", runID)
}

// SkipIfMLflowDisabled skips the current test if the mlflowEnabled flag is false.
func SkipIfMLflowDisabled() {
	ginkgo.GinkgoHelper()
	if !*config.MLflowEnabled {
		ginkgo.Skip("MLflow is not enabled; skipping MLflow integration test")
	}
}

// GetMLflowEndpoint returns the MLflow tracking server endpoint from the
// MLFLOW_TRACKING_URI environment variable.
func GetMLflowEndpoint() string {
	ginkgo.GinkgoHelper()
	endpoint := os.Getenv(mlflowEndpointEnv)
	gomega.Expect(endpoint).NotTo(gomega.BeEmpty(),
		fmt.Sprintf("%s environment variable must be set for MLflow tests", mlflowEndpointEnv))
	return strings.TrimRight(endpoint, "/")
}

// --- KFP Run helpers with plugins_input ---

// CreatePipelineRunWithPluginsInput - Create a pipeline run with plugins_input.
func CreatePipelineRunWithPluginsInput(
	runClient *apiserver.RunClient,
	testContext *apitests.TestContext,
	pipelineID *string,
	pipelineVersionID *string,
	experimentID *string,
	inputParams map[string]interface{},
	pluginsInput map[string]interface{},
) *run_model.V2beta1Run {
	ginkgo.GinkgoHelper()
	runName := fmt.Sprintf("MLflow E2e Test Run-%v", testContext.TestStartTimeUTC)
	runDescription := fmt.Sprintf("MLflow run for %s", runName)
	logger.Log("Create a pipeline run with plugins_input for pipeline id=%s versionId=%s",
		*pipelineID, *pipelineVersionID)

	createRunRequest := &runparams.RunServiceCreateRunParams{
		ExperimentID: experimentID,
		Run: CreatePipelineRunWithPluginsInputPayload(
			runName,
			runDescription,
			pipelineID,
			pipelineVersionID,
			experimentID,
			inputParams,
			pluginsInput,
		),
	}
	createdRun, createRunError := runClient.Create(createRunRequest)
	gomega.Expect(createRunError).NotTo(gomega.HaveOccurred(),
		"Failed to create run with plugins_input for pipeline id="+*pipelineID)
	testContext.PipelineRun.CreatedRunIds = append(testContext.PipelineRun.CreatedRunIds, createdRun.RunID)
	logger.Log("Created Pipeline Run with plugins_input, runId=%s", createdRun.RunID)
	return createdRun
}

// CreatePipelineRunWithPluginsInputPayload - Create a pipeline run payload with plugins_input.
func CreatePipelineRunWithPluginsInputPayload(
	runName string,
	runDescription string,
	pipelineID *string,
	pipelineVersionID *string,
	experimentID *string,
	inputParams map[string]interface{},
	pluginsInput map[string]interface{},
) *run_model.V2beta1Run {
	run := CreatePipelineRunPayload(
		runName,
		runDescription,
		pipelineID,
		pipelineVersionID,
		experimentID,
		inputParams,
	)
	run.PluginsInput = pluginsInput
	return run
}

func BuildMLflowPluginsInput(experimentName string) map[string]interface{} {
	mlflowCfg := map[string]interface{}{}
	if experimentName != "" {
		mlflowCfg["experiment_name"] = experimentName
	}
	return map[string]interface{}{
		mlflowPluginKey: mlflowCfg,
	}
}

func BuildMLflowPluginsInputDisabled() map[string]interface{} {
	return map[string]interface{}{
		mlflowPluginKey: map[string]interface{}{
			"disabled": true,
		},
	}
}

// --- KFP plugins_output verification ---

// VerifyPluginsOutput asserts that the run's plugins_output contains a valid
// MLflow entry with experiment_id, root_run_id, and the expected plugin state.
func VerifyPluginsOutput(run *run_model.V2beta1Run, expectedState run_model.V2beta1PluginState) error {
	ginkgo.GinkgoHelper()
	if run.PluginsOutput == nil {
		return fmt.Errorf("plugins_output should not be nil")
	}
	mlflowOutput, ok := run.PluginsOutput[mlflowPluginKey]
	if !ok {
		return fmt.Errorf("plugins_output should contain %q key", mlflowPluginKey)
	}
	if mlflowOutput.State == nil {
		return fmt.Errorf("plugins_output.%s.state should not be nil", mlflowPluginKey)
	}

	if *mlflowOutput.State != expectedState {
		return fmt.Errorf(
			"plugins_output.%s.state should be %s, got %s",
			mlflowPluginKey, expectedState, *mlflowOutput.State,
		)
	}

	if mlflowOutput.Entries == nil {
		return fmt.Errorf("plugins_output.%s.entries should not be nil", mlflowPluginKey)
	}
	if _, ok := mlflowOutput.Entries["experiment_id"]; !ok {
		return fmt.Errorf("plugins_output.%s.entries should have %q", mlflowPluginKey, "experiment_id")
	}
	if _, ok := mlflowOutput.Entries["root_run_id"]; !ok {
		return fmt.Errorf("plugins_output.%s.entries should have %q", mlflowPluginKey, "root_run_id")
	}
	return nil
}

func GetPluginsOutputEntryValue(run *run_model.V2beta1Run, entryKey string) (string, error) {
	ginkgo.GinkgoHelper()
	if run.PluginsOutput == nil {
		return "", fmt.Errorf("plugins_output should not be nil")
	}
	mlflowOutput, ok := run.PluginsOutput[mlflowPluginKey]
	if !ok {
		return "", fmt.Errorf("plugins_output should contain %q key", mlflowPluginKey)
	}
	entry, ok := mlflowOutput.Entries[entryKey]
	if !ok {
		return "", fmt.Errorf("plugins_output.%s.entries should have %q", mlflowPluginKey, entryKey)
	}
	strVal, ok := entry.Value.(string)
	if !ok {
		return "", fmt.Errorf(
			"plugins_output.%s.entries[%q].value should be a string",
			mlflowPluginKey, entryKey,
		)
	}
	return strVal, nil
}

func VerifyNoPluginsOutput(run *run_model.V2beta1Run) error {
	ginkgo.GinkgoHelper()
	if run.PluginsOutput == nil {
		return nil
	}
	_, ok := run.PluginsOutput[mlflowPluginKey]
	if ok {
		return fmt.Errorf("plugins_output should not contain %q key when MLflow is disabled", mlflowPluginKey)
	}
	return nil
}

type MLflowRun struct {
	Info MLflowRunInfo `json:"info"`
	Data MLflowRunData `json:"data"`
}

type MLflowRunInfo struct {
	RunID        string `json:"run_id"`
	ExperimentID string `json:"experiment_id"`
	Status       string `json:"status"`
}

type MLflowRunData struct {
	Tags []MLflowTag `json:"tags"`
}

type MLflowTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func QueryMLflowExperimentByName(endpoint, experimentName string) (*mlflowclient.MLflowExperiment, error) {
	ginkgo.GinkgoHelper()
	client, err := getMLflowClient(endpoint)
	if err != nil {
		return nil, err
	}
	experiment, err := client.GetExperimentByName(context.Background(), experimentName)
	if err != nil {
		return nil, fmt.Errorf("failed to query MLflow experiment by name %q: %w", experimentName, err)
	}
	return experiment, nil
}

func QueryMLflowRuns(endpoint, experimentID string) ([]MLflowRun, error) {
	return searchMLflowRuns(endpoint, []string{experimentID}, "", 1000)
}

func QueryMLflowRunByID(endpoint, runID string) (*MLflowRun, error) {
	ginkgo.GinkgoHelper()
	client, err := getMLflowClient(endpoint)
	if err != nil {
		return nil, err
	}
	rawRun, err := client.GetRun(context.Background(), runID)
	if err != nil {
		return nil, fmt.Errorf("failed to get MLflow run %q: %w", runID, err)
	}
	var run MLflowRun
	if err := json.Unmarshal(rawRun, &run); err != nil {
		return nil, fmt.Errorf("failed to unmarshal MLflow runs/get response: %w", err)
	}
	return &run, nil
}

func VerifyMLflowRunStatus(endpoint, runID, expectedStatus string) error {
	ginkgo.GinkgoHelper()
	mlflowRun, err := QueryMLflowRunByID(endpoint, runID)
	if err != nil {
		return err
	}
	if mlflowRun.Info.Status != expectedStatus {
		return fmt.Errorf("MLflow run %s should have status %s", runID, expectedStatus)
	}
	return nil
}

func VerifyMLflowRunTags(endpoint, runID string, expectedTags map[string]string) error {
	ginkgo.GinkgoHelper()
	mlflowRun, err := QueryMLflowRunByID(endpoint, runID)
	if err != nil {
		return err
	}
	tagMap := make(map[string]string)
	for _, tag := range mlflowRun.Data.Tags {
		tagMap[tag.Key] = tag.Value
	}
	for key, expectedValue := range expectedTags {
		actualValue, ok := tagMap[key]
		if !ok || actualValue != expectedValue {
			return fmt.Errorf("MLflow run %s should have tag %s=%s", runID, key, expectedValue)
		}
	}
	return nil
}

// QueryNestedRuns returns the MLflow runs whose mlflow.parentRunId tag matches
// the given parent run ID.
func QueryNestedRuns(endpoint, parentRunID string, experimentID ...string) ([]MLflowRun, error) {
	ginkgo.GinkgoHelper()
	filter := fmt.Sprintf(`tags.mlflow.parentRunId = '%s'`, parentRunID)
	if len(experimentID) > 0 && experimentID[0] != "" {
		return searchMLflowRuns(endpoint, []string{experimentID[0]}, filter, 1000)
	}
	return searchMLflowRuns(endpoint, nil, filter, 1000)
}

// CountNestedRuns counts the number of MLflow runs that have a
// mlflow.parentRunId tag matching the given parent run ID.
func CountNestedRuns(endpoint, parentRunID string, experimentID ...string) (int, error) {
	nestedRuns, err := QueryNestedRuns(endpoint, parentRunID, experimentID...)
	if err != nil {
		return 0, err
	}
	return len(nestedRuns), nil
}

func searchMLflowRuns(endpoint string, experimentIDs []string, filter string, maxResults int) ([]MLflowRun, error) {
	ginkgo.GinkgoHelper()
	client, err := getMLflowClient(endpoint)
	if err != nil {
		return nil, err
	}
	pageToken := ""
	var allRuns []MLflowRun
	for {
		response, err := client.SearchRuns(
			context.Background(),
			experimentIDs,
			filter,
			maxResults,
			pageToken,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to search MLflow runs: %w", err)
		}
		for _, rawRun := range response.Runs {
			var parsedRun MLflowRun
			unmarshalErr := json.Unmarshal(rawRun, &parsedRun)
			if unmarshalErr != nil {
				return nil, fmt.Errorf("failed to unmarshal MLflow runs/search response: %w", unmarshalErr)
			}
			allRuns = append(allRuns, parsedRun)
		}
		if response.NextPageToken == "" {
			break
		}
		pageToken = response.NextPageToken
	}
	return allRuns, nil
}
