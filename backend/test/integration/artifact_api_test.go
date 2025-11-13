// Copyright 2025 The Kubeflow Authors
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

package integration

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/golang/glog"
	params "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/experiment_client/experiment_service"
	"github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/experiment_model"
	pipelineParams "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/pipeline_client/pipeline_service"
	uploadParams "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	"github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/pipeline_upload_model"
	runParams "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/run_model"
	"github.com/kubeflow/pipelines/backend/src/common/client/api_server/v1"
	"github.com/kubeflow/pipelines/backend/test"
	"github.com/kubeflow/pipelines/backend/test/config"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ArtifactAPITest struct {
	suite.Suite
	namespace            string
	resourceNamespace    string
	experimentClient     *api_server.ExperimentClient
	pipelineClient       *api_server.PipelineClient
	pipelineUploadClient *api_server.PipelineUploadClient
	runClient            *api_server.RunClient
}

func (s *ArtifactAPITest) SetupTest() {
	if !*runIntegrationTests {
		s.T().SkipNow()
		return
	}

	if !*isDevMode {
		err := test.WaitForReady(*initializeTimeout)
		if err != nil {
			glog.Exitf("Failed to initialize test. Error: %s", err.Error())
		}
	}
	s.namespace = *config.Namespace
	s.resourceNamespace = *config.Namespace
	clientConfig := test.GetClientConfig(*config.Namespace)
	var err error

	s.experimentClient, err = api_server.NewExperimentClient(clientConfig, false)
	if err != nil {
		glog.Exitf("Failed to get experiment client. Error: %v", err)
	}
	s.pipelineClient, err = api_server.NewPipelineClient(clientConfig, false)
	if err != nil {
		glog.Exitf("Failed to get pipeline client. Error: %v", err)
	}
	s.runClient, err = api_server.NewRunClient(clientConfig, false)
	if err != nil {
		glog.Exitf("Failed to get run client. Error: %v", err)
	}
	s.pipelineUploadClient, err = api_server.NewPipelineUploadClient(clientConfig, false)
	if err != nil {
		glog.Exitf("Failed to get pipeline upload client. Error: %v", err)
	}
}

func (s *ArtifactAPITest) TearDownSuite() {
	if *runIntegrationTests {
		if !*isDevMode {
			s.cleanUp()
		}
	}
}

func (s *ArtifactAPITest) TearDownTest() {
	if *runIntegrationTests {
		s.cleanUp()
	}
}

func (s *ArtifactAPITest) cleanUp() {
	test.DeleteAllRuns(s.runClient, s.namespace, s.T())
	test.DeleteAllExperiments(s.experimentClient, s.namespace, s.T())
	test.DeleteAllPipelines(s.pipelineClient, s.T())
}

func (s *ArtifactAPITest) TestV1PipelineArtifactRead() {
	runID, experimentID, pipelineID := s.runPipeline()

	defer func() {
		s.deleteRun(runID)
		s.deleteExperiment(experimentID)
		s.deleteAllPipelineVersions(pipelineID)
	}()

	s.waitForRunCompletion(runID)

	nodeID := s.extractWorkflowNodeID(runID)
	artifactName := "generate-large-artifact-large_file"

	s.testReadArtifactEndpoint(runID, nodeID, artifactName)
}

func (s *ArtifactAPITest) deleteAllPipelineVersions(pipelineID string) {
	test.DeleteAllPipelineVersions(s.pipelineClient, s.T(), pipelineID)
	if err := s.pipelineClient.Delete(&pipelineParams.PipelineServiceDeletePipelineV1Params{
		ID: pipelineID,
	}); err != nil {
		s.T().Logf("Failed to clean up test pipeline %s: %v", pipelineID, err)
	}
}

func (s *ArtifactAPITest) deleteExperiment(experimentID string) {
	if err := s.experimentClient.Delete(&params.ExperimentServiceDeleteExperimentV1Params{
		ID: experimentID,
	}); err != nil {
		s.T().Logf("Failed to clean up test experiment %s: %v", experimentID, err)
	}
}

func (s *ArtifactAPITest) deleteRun(runID string) {
	if err := s.runClient.Delete(&runParams.RunServiceDeleteRunV1Params{
		ID: runID,
	}); err != nil {
		s.T().Logf("Failed to clean up test run %s: %v", runID, err)
	}
}

func (s *ArtifactAPITest) createExperiment() *experiment_model.APIExperiment {
	experimentName := fmt.Sprintf("artifact-test-experiment-%d", time.Now().Unix())
	experiment := test.GetExperiment(experimentName, "Test for artifact reading", s.namespace)
	experiment, err := s.experimentClient.Create(&params.ExperimentServiceCreateExperimentV1Params{Experiment: experiment})
	require.NoError(s.T(), err)
	return experiment
}

func (s *ArtifactAPITest) uploadPipeline() *pipeline_upload_model.APIPipeline {
	pipelineParams := uploadParams.NewUploadPipelineParams()
	pipelineName := fmt.Sprintf("large-artifact-test-%d", time.Now().Unix())
	pipelineParams.SetName(&pipelineName)
	pipeline, err := s.pipelineUploadClient.UploadFile("../resources/large_artifact_pipeline.yaml", pipelineParams)
	require.NoError(s.T(), err, "Failed to upload pipeline")
	return pipeline
}

func (s *ArtifactAPITest) runPipeline() (runID, experimentID, pipelineID string) {
	experiment := s.createExperiment()
	pipeline := s.uploadPipeline()

	runRequest := &runParams.RunServiceCreateRunV1Params{Run: &run_model.APIRun{
		Name: fmt.Sprintf("test-artifact-run-%d", time.Now().Unix()),
		PipelineSpec: &run_model.APIPipelineSpec{
			PipelineID: pipeline.ID,
		},
		ResourceReferences: []*run_model.APIResourceReference{
			{
				Key: &run_model.APIResourceKey{
					Type: run_model.APIResourceTypeEXPERIMENT.Pointer(),
					ID:   experiment.ID,
				},
				Relationship: run_model.APIRelationshipOWNER.Pointer(),
			},
		},
	}}
	run, _, err := s.runClient.Create(runRequest)
	require.NoError(s.T(), err)
	return run.Run.ID, experiment.ID, pipeline.ID
}

func (s *ArtifactAPITest) waitForRunCompletion(runID string) {
	t := s.T()
	maxWaitTime := 2 * time.Minute
	startTime := time.Now()

	for time.Since(startTime) < maxWaitTime {
		runDetail, _, err := s.runClient.Get(&runParams.RunServiceGetRunV1Params{RunID: runID})
		require.NoError(t, err)

		if runDetail.Run.Status == "Succeeded" || runDetail.Run.Status == "Completed" ||
			runDetail.Run.Status == "Failed" || runDetail.Run.Status == "Error" {
			require.Contains(t, []string{"Succeeded", "Completed"}, runDetail.Run.Status,
				"Run should have succeeded")
			t.Log("Run completed")
			return
		}

		time.Sleep(2 * time.Second)
	}

	require.Fail(t, "Run did not complete within %v", maxWaitTime)
}

func (s *ArtifactAPITest) extractWorkflowNodeID(runID string) string {
	t := s.T()

	resp, err := http.Get(fmt.Sprintf("http://localhost:8888/apis/v1beta1/runs/%s", runID))
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)

	runDetailsStr := string(body)
	re := regexp.MustCompile(`large-artifact-memory-test-v1-1gb-[a-z0-9]+-[0-9]+`)
	matches := re.FindAllString(runDetailsStr, -1)

	require.NotEmpty(t, matches, "Could not find workflow name in run details")

	nodeID := matches[0]
	t.Logf("Found workflow name: %s", nodeID)

	return nodeID
}

func (s *ArtifactAPITest) testReadArtifactEndpoint(runID, nodeID, artifactName string) {
	t := s.T()

	baseURL := "http://localhost:8888"
	artifactURL := fmt.Sprintf("%s/apis/v1beta1/runs/%s/nodes/%s/artifacts/%s:read",
		baseURL, runID, nodeID, artifactName)

	t.Logf("Testing ReadArtifact endpoint: %s", artifactURL)

	resp, err := http.Get(artifactURL)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode,
		"ReadArtifact endpoint should return 200 OK, got %d", resp.StatusCode)

	s.validateResponseHeaders(resp)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NotEmpty(t, body)

	// The response is JSON with base64-encoded gzip data
	var jsonResponse struct {
		Data string `json:"data"`
	}
	err = json.Unmarshal(body, &jsonResponse)
	require.NoError(t, err, "Failed to parse JSON response")
	require.NotEmpty(t, jsonResponse.Data, "JSON response should contain 'data' field")

	decodedData, err := base64.StdEncoding.DecodeString(jsonResponse.Data)
	require.NoError(t, err, "Failed to decode base64 data")
	require.NotEmpty(t, decodedData)

	s.requireGzipCompressed(decodedData)

	t.Logf("Successfully downloaded gzip compressed artifact (decoded size: %d bytes)", len(decodedData))
	t.Log("ReadArtifact endpoint validated successfully")
}

func (s *ArtifactAPITest) validateResponseHeaders(resp *http.Response) {
	t := s.T()

	contentType := resp.Header.Get("Content-Type")
	require.Equal(t, "application/json", contentType,
		"Content-Type should be application/json, got %s", contentType)

	contentEncoding := resp.Header.Get("Content-Encoding")
	require.Empty(t, contentEncoding,
		"Content-Encoding should not be set for JSON response (gzip is base64-encoded in JSON)")

	t.Log("Response headers:")
	for key, values := range resp.Header {
		for _, value := range values {
			t.Logf("  %s: %s", key, value)
		}
	}
}

func (s *ArtifactAPITest) requireGzipCompressed(body []byte) {
	require.True(s.T(), len(body) > 2 && body[0] == 0x1f && body[1] == 0x8b,
		"Artifact should be gzip compressed")
}

func TestArtifactAPI(t *testing.T) {
	suite.Run(t, new(ArtifactAPITest))
}
