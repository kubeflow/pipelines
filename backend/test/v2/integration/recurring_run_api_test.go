// Copyright 2018-2023 The Kubeflow Authors
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
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/eapache/go-resiliency/retrier"
	"github.com/go-openapi/strfmt"
	"github.com/golang/glog"
	experiment_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_client/experiment_service"
	params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_client/pipeline_service"
	upload_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	recurring_run_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/recurring_run_client/recurring_run_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/recurring_run_model"
	run_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	api_server "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v2"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	test "github.com/kubeflow/pipelines/backend/test/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/types/known/structpb"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

const (
	second = 1
	minute = 60 * second
	hour   = 60 * minute
)

type RecurringRunApiTestSuite struct {
	suite.Suite
	namespace            string
	resourceNamespace    string
	experimentClient     *api_server.ExperimentClient
	pipelineClient       *api_server.PipelineClient
	pipelineUploadClient *api_server.PipelineUploadClient
	runClient            *api_server.RunClient
	recurringRunClient   *api_server.RecurringRunClient
	swfClient            client.SwfClientInterface
}

// Check the namespace have ML pipeline installed and ready
func (s *RecurringRunApiTestSuite) SetupTest() {
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
	s.namespace = *namespace

	var newExperimentClient func() (*api_server.ExperimentClient, error)
	var newPipelineUploadClient func() (*api_server.PipelineUploadClient, error)
	var newPipelineClient func() (*api_server.PipelineClient, error)
	var newRunClient func() (*api_server.RunClient, error)
	var newRecurringRunClient func() (*api_server.RecurringRunClient, error)

	if *isKubeflowMode {
		s.resourceNamespace = *resourceNamespace

		newExperimentClient = func() (*api_server.ExperimentClient, error) {
			return api_server.NewKubeflowInClusterExperimentClient(s.namespace, *isDebugMode)
		}
		newPipelineUploadClient = func() (*api_server.PipelineUploadClient, error) {
			return api_server.NewKubeflowInClusterPipelineUploadClient(s.namespace, *isDebugMode)
		}
		newPipelineClient = func() (*api_server.PipelineClient, error) {
			return api_server.NewKubeflowInClusterPipelineClient(s.namespace, *isDebugMode)
		}
		newRunClient = func() (*api_server.RunClient, error) {
			return api_server.NewKubeflowInClusterRunClient(s.namespace, *isDebugMode)
		}
		newRecurringRunClient = func() (*api_server.RecurringRunClient, error) {
			return api_server.NewKubeflowInClusterRecurringRunClient(s.namespace, *isDebugMode)
		}
	} else {
		clientConfig := test.GetClientConfig(*namespace)

		newExperimentClient = func() (*api_server.ExperimentClient, error) {
			return api_server.NewExperimentClient(clientConfig, *isDebugMode)
		}
		newPipelineUploadClient = func() (*api_server.PipelineUploadClient, error) {
			return api_server.NewPipelineUploadClient(clientConfig, *isDebugMode)
		}
		newPipelineClient = func() (*api_server.PipelineClient, error) {
			return api_server.NewPipelineClient(clientConfig, *isDebugMode)
		}
		newRunClient = func() (*api_server.RunClient, error) {
			return api_server.NewRunClient(clientConfig, *isDebugMode)
		}
		newRecurringRunClient = func() (*api_server.RecurringRunClient, error) {
			return api_server.NewRecurringRunClient(clientConfig, *isDebugMode)
		}
	}

	var err error
	s.experimentClient, err = newExperimentClient()
	if err != nil {
		glog.Exitf("Failed to get experiment client. Error: %v", err)
	}
	s.pipelineUploadClient, err = newPipelineUploadClient()
	if err != nil {
		glog.Exitf("Failed to get pipeline upload client. Error: %s", err.Error())
	}
	s.pipelineClient, err = newPipelineClient()
	if err != nil {
		glog.Exitf("Failed to get pipeline client. Error: %s", err.Error())
	}
	s.runClient, err = newRunClient()
	if err != nil {
		glog.Exitf("Failed to get run client. Error: %s", err.Error())
	}
	s.recurringRunClient, err = newRecurringRunClient()
	if err != nil {
		glog.Exitf("Failed to get recurringRun client. Error: %s", err.Error())
	}
	s.swfClient = client.NewScheduledWorkflowClientOrFatal(time.Second*30, util.ClientParameters{QPS: 5, Burst: 10})

	s.cleanUp()
}

func (s *RecurringRunApiTestSuite) TestRecurringRunApis() {
	t := s.T()

	/* ---------- Upload pipelines YAML ---------- */
	helloWorldPipeline, err := s.pipelineUploadClient.UploadFile("../resources/hello-world.yaml", upload_params.NewUploadPipelineParams())
	assert.Nil(t, err)

	/* ---------- Upload pipeline version YAML ---------- */
	time.Sleep(1 * time.Second)
	helloWorldPipelineVersion, err := s.pipelineUploadClient.UploadPipelineVersion(
		"../resources/hello-world.yaml", &upload_params.UploadPipelineVersionParams{
			Name:       util.StringPointer("hello-world-version"),
			Pipelineid: util.StringPointer(helloWorldPipeline.PipelineID),
		})
	assert.Nil(t, err)

	/* ---------- Create a new hello world experiment ---------- */
	experiment := test.MakeExperiment("hello world experiment", "", s.resourceNamespace)
	helloWorldExperiment, err := s.experimentClient.Create(&experiment_params.ExperimentServiceCreateExperimentParams{Body: experiment})
	assert.Nil(t, err)

	/* ---------- Create a new hello world recurringRun by specifying pipeline ID ---------- */
	createRecurringRunRequest := &recurring_run_params.RecurringRunServiceCreateRecurringRunParams{Body: &recurring_run_model.V2beta1RecurringRun{
		DisplayName:  "hello world",
		Description:  "this is hello world",
		ExperimentID: helloWorldExperiment.ExperimentID,
		PipelineVersionReference: &recurring_run_model.V2beta1PipelineVersionReference{
			PipelineID:        helloWorldPipelineVersion.PipelineID,
			PipelineVersionID: helloWorldPipelineVersion.PipelineVersionID,
		},
		MaxConcurrency: 10,
		Mode:           recurring_run_model.RecurringRunModeENABLE,
	}}
	helloWorldRecurringRun, err := s.recurringRunClient.Create(createRecurringRunRequest)
	assert.Nil(t, err)
	s.checkHelloWorldRecurringRun(t, helloWorldRecurringRun, helloWorldExperiment.ExperimentID, helloWorldPipelineVersion.PipelineID, helloWorldPipelineVersion.PipelineVersionID)

	/* ---------- Get hello world recurringRun ---------- */
	helloWorldRecurringRun, err = s.recurringRunClient.Get(&recurring_run_params.RecurringRunServiceGetRecurringRunParams{RecurringRunID: helloWorldRecurringRun.RecurringRunID})
	assert.Nil(t, err)
	s.checkHelloWorldRecurringRun(t, helloWorldRecurringRun, helloWorldExperiment.ExperimentID, helloWorldPipelineVersion.PipelineID, helloWorldPipelineVersion.PipelineVersionID)

	/* ---------- Create a new argument parameter experiment ---------- */
	experiment = test.MakeExperiment("argument parameter experiment", "", s.resourceNamespace)
	argParamsExperiment, err := s.experimentClient.Create(&experiment_params.ExperimentServiceCreateExperimentParams{Body: experiment})
	assert.Nil(t, err)

	/* ---------- Create a new argument parameter recurringRun by uploading workflow manifest ---------- */
	// Make sure the recurringRun is created at least 1 second later than the first one,
	// because sort by created_at has precision of 1 second.
	time.Sleep(1 * time.Second)
	argParamsBytes, err := ioutil.ReadFile("../resources/arguments-parameters.yaml")
	assert.Nil(t, err)
	pipeline_spec := &structpb.Struct{}
	err = yaml.Unmarshal(argParamsBytes, pipeline_spec)
	assert.Nil(t, err)

	createRecurringRunRequest = &recurring_run_params.RecurringRunServiceCreateRecurringRunParams{Body: &recurring_run_model.V2beta1RecurringRun{
		DisplayName:  "argument parameter",
		Description:  "this is argument parameter",
		ExperimentID: argParamsExperiment.ExperimentID,
		PipelineSpec: pipeline_spec,
		RuntimeConfig: &recurring_run_model.V2beta1RuntimeConfig{
			Parameters: map[string]interface{}{
				"param1": "goodbye",
				"param2": "world",
			},
		},
		MaxConcurrency: 10,
		Mode:           recurring_run_model.RecurringRunModeENABLE,
	}}
	argParamsRecurringRun, err := s.recurringRunClient.Create(createRecurringRunRequest)
	assert.Nil(t, err)
	s.checkArgParamsRecurringRun(t, argParamsRecurringRun, argParamsExperiment.ExperimentID)

	/* ---------- List all the recurringRuns. Both recurringRuns should be returned ---------- */
	recurringRuns, totalSize, _, err := test.ListAllRecurringRuns(s.recurringRunClient, s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 2, totalSize, "Incorrect total number of recurring runs in the namespace %v", s.resourceNamespace)
	assert.Equal(t, 2, len(recurringRuns), "Incorrect total number of recurring runs in the namespace %v", s.resourceNamespace)

	/* ---------- List the recurringRuns, paginated, sort by creation time ---------- */
	recurringRuns, totalSize, nextPageToken, err := test.ListRecurringRuns(
		s.recurringRunClient,
		&recurring_run_params.RecurringRunServiceListRecurringRunsParams{
			PageSize: util.Int32Pointer(1),
			SortBy:   util.StringPointer("created_at"),
		},
		s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(recurringRuns))
	assert.Equal(t, 2, totalSize)
	assert.Equal(t, "hello world", recurringRuns[0].DisplayName)
	recurringRuns, totalSize, _, err = test.ListRecurringRuns(
		s.recurringRunClient,
		&recurring_run_params.RecurringRunServiceListRecurringRunsParams{
			PageSize:  util.Int32Pointer(1),
			PageToken: util.StringPointer(nextPageToken),
		},
		s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(recurringRuns))
	assert.Equal(t, 2, totalSize)
	assert.Equal(t, "argument parameter", recurringRuns[0].DisplayName)

	/* ---------- List the recurringRuns, paginated, sort by name ---------- */
	recurringRuns, totalSize, nextPageToken, err = test.ListRecurringRuns(
		s.recurringRunClient,
		&recurring_run_params.RecurringRunServiceListRecurringRunsParams{
			PageSize: util.Int32Pointer(1),
			SortBy:   util.StringPointer("name"),
		},
		s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 2, totalSize)
	assert.Equal(t, 1, len(recurringRuns))
	assert.Equal(t, "argument parameter", recurringRuns[0].DisplayName)
	recurringRuns, totalSize, _, err = test.ListRecurringRuns(
		s.recurringRunClient,
		&recurring_run_params.RecurringRunServiceListRecurringRunsParams{
			PageSize:  util.Int32Pointer(1),
			SortBy:    util.StringPointer("name"),
			PageToken: util.StringPointer(nextPageToken),
		},
		s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 2, totalSize)
	assert.Equal(t, 1, len(recurringRuns))
	assert.Equal(t, "hello world", recurringRuns[0].DisplayName)

	/* ---------- List the recurringRuns, sort by unsupported field ---------- */
	recurringRuns, _, _, err = test.ListRecurringRuns(
		s.recurringRunClient,
		&recurring_run_params.RecurringRunServiceListRecurringRunsParams{
			PageSize: util.Int32Pointer(2),
			SortBy:   util.StringPointer("unknown"),
		},
		s.resourceNamespace)
	assert.NotNil(t, err)
	assert.Equal(t, len(recurringRuns), 0)

	/* ---------- List recurringRuns for hello world experiment. One recurringRun should be returned ---------- */
	recurringRuns, totalSize, _, err = s.recurringRunClient.List(&recurring_run_params.RecurringRunServiceListRecurringRunsParams{
		ExperimentID: util.StringPointer(helloWorldExperiment.ExperimentID),
	})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(recurringRuns))
	assert.Equal(t, 1, totalSize)
	assert.Equal(t, "hello world", recurringRuns[0].DisplayName)

	/* ---------- List the recurringRuns, filtered by created_at, only return the previous two recurringRuns ---------- */
	time.Sleep(5 * time.Second) // Sleep for 5 seconds to make sure the previous recurringRuns are created at a different timestamp
	filterTime := time.Now().Unix()
	time.Sleep(5 * time.Second)
	createRecurringRunRequestNew := &recurring_run_params.RecurringRunServiceCreateRecurringRunParams{Body: &recurring_run_model.V2beta1RecurringRun{
		DisplayName:  "new hello world recurringRun",
		Description:  "this is a new hello world",
		ExperimentID: helloWorldExperiment.ExperimentID,
		PipelineVersionReference: &recurring_run_model.V2beta1PipelineVersionReference{
			PipelineID:        helloWorldPipelineVersion.PipelineID,
			PipelineVersionID: helloWorldPipelineVersion.PipelineVersionID,
		},
		MaxConcurrency: 10,
		Mode:           recurring_run_model.RecurringRunModeDISABLE,
	}}
	_, err = s.recurringRunClient.Create(createRecurringRunRequestNew)
	assert.Nil(t, err)
	// Check total number of recurringRuns to be 3
	recurringRuns, totalSize, _, err = test.ListAllRecurringRuns(s.recurringRunClient, s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 3, totalSize)
	assert.Equal(t, 3, len(recurringRuns))
	// Check number of filtered recurringRuns finished before filterTime to be 2
	recurringRuns, totalSize, _, err = test.ListRecurringRuns(
		s.recurringRunClient,
		&recurring_run_params.RecurringRunServiceListRecurringRunsParams{
			Filter: util.StringPointer(`{"predicates": [{"key": "created_at", "operation": "LESS_THAN", "string_value": "` + fmt.Sprint(filterTime) + `"}]}`),
		},
		s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(recurringRuns))
	assert.Equal(t, 2, totalSize)

	// The scheduledWorkflow CRD would create the run and it synced to the DB by persistent agent.
	// This could take a few seconds to finish.

	/* ---------- Check run for hello world recurringRun ---------- */
	if err := retrier.New(retrier.ConstantBackoff(8, 5*time.Second), nil).Run(func() error {
		runs, totalSize, _, err := s.runClient.List(&run_params.RunServiceListRunsParams{
			ExperimentID: util.StringPointer(helloWorldExperiment.ExperimentID),
		})
		if err != nil {
			return err
		}
		if len(runs) != 1 {
			return fmt.Errorf("expected runs to be length 1, got: %v", len(runs))
		}
		if totalSize != 1 {
			return fmt.Errorf("expected total size 1, got: %v", totalSize)
		}
		helloWorldRun := runs[0]
		return s.checkHelloWorldRun(helloWorldRun, helloWorldExperiment.ExperimentID, helloWorldRecurringRun.RecurringRunID)
	}); err != nil {
		assert.Nil(t, err)
	}

	/* ---------- Check run for argument parameter recurringRun ---------- */
	if err := retrier.New(retrier.ConstantBackoff(8, 5*time.Second), nil).Run(func() error {
		runs, totalSize, _, err := s.runClient.List(&run_params.RunServiceListRunsParams{
			ExperimentID: util.StringPointer(argParamsExperiment.ExperimentID),
		})
		if err != nil {
			return err
		}
		if len(runs) != 1 {
			return fmt.Errorf("expected runs to be length 1, got: %v", len(runs))
		}
		if totalSize != 1 {
			return fmt.Errorf("expected total size 1, got: %v", totalSize)
		}
		argParamsRun := runs[0]
		return s.checkArgParamsRun(argParamsRun, argParamsExperiment.ExperimentID, argParamsRecurringRun.RecurringRunID)
	}); err != nil {
		assert.Nil(t, err)
	}
}

func (s *RecurringRunApiTestSuite) TestRecurringRunApis_noCatchupOption() {
	t := s.T()

	/* ---------- Upload pipelines YAML ---------- */
	pipeline, err := s.pipelineUploadClient.UploadFile("../resources/hello-world.yaml", upload_params.NewUploadPipelineParams())
	assert.Nil(t, err)

	/* ---------- Upload pipeline version YAML ---------- */
	time.Sleep(1 * time.Second)
	helloWorldPipelineVersion, err := s.pipelineUploadClient.UploadPipelineVersion(
		"../resources/hello-world.yaml", &upload_params.UploadPipelineVersionParams{
			Name:       util.StringPointer("hello-world-version"),
			Pipelineid: util.StringPointer(pipeline.PipelineID),
		})
	assert.Nil(t, err)

	/* ---------- Create a periodic recurringRun with start and end date in the past and catchup = true ---------- */
	experiment := test.MakeExperiment("periodic catchup true", "", s.resourceNamespace)
	periodicCatchupTrueExperiment, err := s.experimentClient.Create(&experiment_params.ExperimentServiceCreateExperimentParams{Body: experiment})
	assert.Nil(t, err)

	recurringRun := recurringRunInThePastForTwoMinutes(recurringRunOptions{
		pipelineId:        helloWorldPipelineVersion.PipelineID,
		pipelineVersionId: helloWorldPipelineVersion.PipelineVersionID,
		experimentId:      periodicCatchupTrueExperiment.ExperimentID,
		periodic:          true,
	})
	recurringRun.DisplayName = "periodic-catchup-true-"
	recurringRun.Description = "A recurringRun with NoCatchup=false will backfill each past interval when behind schedule."
	recurringRun.NoCatchup = false // This is the key difference.
	createRecurringRunRequest := &recurring_run_params.RecurringRunServiceCreateRecurringRunParams{Body: recurringRun}
	_, err = s.recurringRunClient.Create(createRecurringRunRequest)
	assert.Nil(t, err)

	/* -------- Create another periodic recurringRun with start and end date in the past but catchup = false ------ */
	experiment = test.MakeExperiment("periodic catchup false", "", s.resourceNamespace)
	periodicCatchupFalseExperiment, err := s.experimentClient.Create(&experiment_params.ExperimentServiceCreateExperimentParams{Body: experiment})
	assert.Nil(t, err)

	recurringRun = recurringRunInThePastForTwoMinutes(recurringRunOptions{
		pipelineId:        helloWorldPipelineVersion.PipelineID,
		pipelineVersionId: helloWorldPipelineVersion.PipelineVersionID,
		experimentId:      periodicCatchupFalseExperiment.ExperimentID,
		periodic:          true,
	})
	recurringRun.DisplayName = "periodic-catchup-false-"
	recurringRun.Description = "A recurringRun with NoCatchup=true only schedules the last interval when behind schedule."
	recurringRun.NoCatchup = true // This is the key difference.
	createRecurringRunRequest = &recurring_run_params.RecurringRunServiceCreateRecurringRunParams{Body: recurringRun}
	_, err = s.recurringRunClient.Create(createRecurringRunRequest)
	assert.Nil(t, err)

	/* ---------- Create a cron recurringRun with start and end date in the past and catchup = true ---------- */
	experiment = test.MakeExperiment("cron catchup true", "", s.resourceNamespace)
	cronCatchupTrueExperiment, err := s.experimentClient.Create(&experiment_params.ExperimentServiceCreateExperimentParams{Body: experiment})
	assert.Nil(t, err)

	recurringRun = recurringRunInThePastForTwoMinutes(recurringRunOptions{
		pipelineId:        helloWorldPipelineVersion.PipelineID,
		pipelineVersionId: helloWorldPipelineVersion.PipelineVersionID,
		experimentId:      cronCatchupTrueExperiment.ExperimentID,
		periodic:          false,
	})
	recurringRun.DisplayName = "cron-catchup-true-"
	recurringRun.Description = "A recurringRun with NoCatchup=false will backfill each past interval when behind schedule."
	recurringRun.NoCatchup = false // This is the key difference.
	createRecurringRunRequest = &recurring_run_params.RecurringRunServiceCreateRecurringRunParams{Body: recurringRun}
	_, err = s.recurringRunClient.Create(createRecurringRunRequest)
	assert.Nil(t, err)

	/* -------- Create another cron recurringRun with start and end date in the past but catchup = false ------ */
	experiment = test.MakeExperiment("cron catchup false", "", s.resourceNamespace)
	cronCatchupFalseExperiment, err := s.experimentClient.Create(&experiment_params.ExperimentServiceCreateExperimentParams{Body: experiment})
	assert.Nil(t, err)

	recurringRun = recurringRunInThePastForTwoMinutes(recurringRunOptions{
		pipelineId:        helloWorldPipelineVersion.PipelineID,
		pipelineVersionId: helloWorldPipelineVersion.PipelineVersionID,
		experimentId:      cronCatchupFalseExperiment.ExperimentID,
		periodic:          false,
	})
	recurringRun.DisplayName = "cron-catchup-false-"
	recurringRun.Description = "A recurringRun with NoCatchup=true only schedules the last interval when behind schedule."
	recurringRun.NoCatchup = true // This is the key difference.
	createRecurringRunRequest = &recurring_run_params.RecurringRunServiceCreateRecurringRunParams{Body: recurringRun}
	_, err = s.recurringRunClient.Create(createRecurringRunRequest)
	assert.Nil(t, err)

	// The scheduledWorkflow CRD would create the run and it is synced to the DB by persistent agent.
	// This could take a few seconds to finish.

	/* ---------- Assert number of runs when catchup = true ---------- */
	if err := retrier.New(retrier.ConstantBackoff(8, 5*time.Second), nil).Run(func() error {
		_, runsWhenCatchupTrue, _, err := s.runClient.List(&run_params.RunServiceListRunsParams{
			ExperimentID: util.StringPointer(periodicCatchupTrueExperiment.ExperimentID),
		})
		if err != nil {
			return err
		}
		if runsWhenCatchupTrue != 2 {
			return fmt.Errorf("expected runsWhenCatchupTrue with periodic schedule to be 2, got: %v", runsWhenCatchupTrue)
		}

		_, runsWhenCatchupTrue, _, err = s.runClient.List(&run_params.RunServiceListRunsParams{
			ExperimentID: util.StringPointer(cronCatchupTrueExperiment.ExperimentID),
		})
		if err != nil {
			return err
		}
		if runsWhenCatchupTrue != 2 {
			return fmt.Errorf("expected runsWhenCatchupTrue with cron schedule to be 2, got: %v", runsWhenCatchupTrue)
		}

		return nil
	}); err != nil {
		assert.Nil(t, err)
	}

	/* ---------- Assert number of runs when catchup = false ---------- */
	if err := retrier.New(retrier.ConstantBackoff(8, 5*time.Second), nil).Run(func() error {
		_, runsWhenCatchupFalse, _, err := s.runClient.List(&run_params.RunServiceListRunsParams{
			ExperimentID: util.StringPointer(periodicCatchupFalseExperiment.ExperimentID),
		})
		if err != nil {
			return err
		}
		if runsWhenCatchupFalse != 1 {
			return fmt.Errorf("expected runsWhenCatchupFalse with periodic schedule to be 1, got: %v", runsWhenCatchupFalse)
		}

		_, runsWhenCatchupFalse, _, err = s.runClient.List(&run_params.RunServiceListRunsParams{
			ExperimentID: util.StringPointer(cronCatchupFalseExperiment.ExperimentID),
		})
		if err != nil {
			return err
		}
		if runsWhenCatchupFalse != 1 {
			return fmt.Errorf("expected runsWhenCatchupFalse with cron schedule to be 1, got: %v", runsWhenCatchupFalse)
		}
		return nil
	}); err != nil {
		assert.Nil(t, err)
	}
}

func (s *RecurringRunApiTestSuite) checkHelloWorldRecurringRun(t *testing.T, recurringRun *recurring_run_model.V2beta1RecurringRun, experimentID string, pipelineId string, pipelineVersionId string) {
	expectedRecurringRun := &recurring_run_model.V2beta1RecurringRun{
		RecurringRunID: recurringRun.RecurringRunID,
		DisplayName:    "hello world",
		Description:    "this is hello world",
		ServiceAccount: test.GetDefaultPipelineRunnerServiceAccount(*isKubeflowMode),
		PipelineSpec:   recurringRun.PipelineSpec,
		ExperimentID:   experimentID,
		PipelineVersionReference: &recurring_run_model.V2beta1PipelineVersionReference{
			PipelineID:        pipelineId,
			PipelineVersionID: pipelineVersionId,
		},
		MaxConcurrency: 10,
		Mode:           recurring_run_model.RecurringRunModeENABLE,
		Namespace:      recurringRun.Namespace,
		CreatedAt:      recurringRun.CreatedAt,
		UpdatedAt:      recurringRun.UpdatedAt,
		Trigger:        recurringRun.Trigger,
		Status:         recurringRun.Status,
	}
	assert.Equal(t, expectedRecurringRun, recurringRun)
}

func (s *RecurringRunApiTestSuite) checkArgParamsRecurringRun(t *testing.T, recurringRun *recurring_run_model.V2beta1RecurringRun, experimentID string) {
	expectedRecurringRun := &recurring_run_model.V2beta1RecurringRun{
		RecurringRunID: recurringRun.RecurringRunID,
		DisplayName:    "argument parameter",
		Description:    "this is argument parameter",
		ServiceAccount: test.GetDefaultPipelineRunnerServiceAccount(*isKubeflowMode),
		PipelineSpec:   recurringRun.PipelineSpec,
		RuntimeConfig: &recurring_run_model.V2beta1RuntimeConfig{
			Parameters: map[string]interface{}{
				"param1": "goodbye",
				"param2": "world",
			},
		},
		ExperimentID:   experimentID,
		MaxConcurrency: 10,
		Mode:           recurring_run_model.RecurringRunModeENABLE,
		Namespace:      recurringRun.Namespace,
		CreatedAt:      recurringRun.CreatedAt,
		UpdatedAt:      recurringRun.UpdatedAt,
		Trigger:        recurringRun.Trigger,
		Status:         recurringRun.Status,
	}
	assert.Equal(t, expectedRecurringRun, recurringRun)
}

func (s *RecurringRunApiTestSuite) TestRecurringRunApis_SwfNotFound() {
	t := s.T()

	/* ---------- Upload pipelines YAML ---------- */
	pipeline, err := s.pipelineUploadClient.UploadFile("../resources/hello-world.yaml", upload_params.NewUploadPipelineParams())
	assert.Nil(t, err)
	pipelineVersions, totalSize, _, err := s.pipelineClient.ListPipelineVersions(&params.PipelineServiceListPipelineVersionsParams{
		PipelineID: pipeline.PipelineID,
	})
	assert.Nil(t, err)
	assert.Equal(t, totalSize, 1)

	/* ---------- Create a new hello world recurringRun by specifying pipeline ID ---------- */
	experiment := test.MakeExperiment("test-swf-not-found experiment", "", s.resourceNamespace)
	swfNotFoundExperiment, err := s.experimentClient.Create(&experiment_params.ExperimentServiceCreateExperimentParams{Body: experiment})
	assert.Nil(t, err)

	createRecurringRunRequest := &recurring_run_params.RecurringRunServiceCreateRecurringRunParams{Body: &recurring_run_model.V2beta1RecurringRun{
		DisplayName:  "test-swf-not-found",
		ExperimentID: swfNotFoundExperiment.ExperimentID,
		PipelineVersionReference: &recurring_run_model.V2beta1PipelineVersionReference{
			PipelineID:        pipelineVersions[0].PipelineID,
			PipelineVersionID: pipelineVersions[0].PipelineVersionID,
		},
		MaxConcurrency: 10,
		Mode:           recurring_run_model.RecurringRunModeDISABLE,
	}}

	recurringRun, err := s.recurringRunClient.Create(createRecurringRunRequest)
	assert.Nil(t, err)

	// Delete all ScheduledWorkflow custom resources to simulate the situation
	// that after reinstalling KFP with managed storage, only KFP DB is kept,
	// but all KFP custom resources are gone.
	swfNamespace := s.namespace
	if s.resourceNamespace != "" {
		swfNamespace = s.resourceNamespace
	}
	err = s.swfClient.ScheduledWorkflow(swfNamespace).DeleteCollection(context.Background(), &v1.DeleteOptions{}, v1.ListOptions{})
	assert.Nil(t, err)

	err = s.recurringRunClient.Delete(&recurring_run_params.RecurringRunServiceDeleteRecurringRunParams{RecurringRunID: recurringRun.RecurringRunID})
	assert.Nil(t, err)

	/* ---------- Get recurringRun ---------- */
	_, err = s.recurringRunClient.Get(&recurring_run_params.RecurringRunServiceGetRecurringRunParams{RecurringRunID: recurringRun.RecurringRunID})
	assert.NotNil(t, err)
	// Check the error contains a 404 (not found) status code
	assert.Contains(t, err.Error(), "[404]")
}

func (s *RecurringRunApiTestSuite) checkHelloWorldRun(run *run_model.V2beta1Run, experimentID string, recurringRunID string) error {
	if !strings.Contains(run.DisplayName, "helloworld") {
		return fmt.Errorf("expected: %+v got: %+v", "helloworld", run.DisplayName)
	}

	if run.ExperimentID != experimentID {
		return fmt.Errorf("expected: %+v got: %+v", experimentID, run.ExperimentID)
	}

	if run.RecurringRunID != recurringRunID {
		return fmt.Errorf("expected: %+v got: %+v", recurringRunID, run.RecurringRunID)
	}

	return nil
}

func (s *RecurringRunApiTestSuite) checkArgParamsRun(run *run_model.V2beta1Run, experimentID, recurringRunID string) error {
	if !strings.Contains(run.DisplayName, "argumentparameter") {
		return fmt.Errorf("expected: %+v got: %+v", "argumentparameter", run.DisplayName)
	}
	if run.ExperimentID != experimentID {
		return fmt.Errorf("expected: %+v got: %+v", experimentID, run.ExperimentID)
	}

	if run.RecurringRunID != recurringRunID {
		return fmt.Errorf("expected: %+v got: %+v", recurringRunID, run.RecurringRunID)
	}
	return nil
}

func TestRecurringRunApi(t *testing.T) {
	suite.Run(t, new(RecurringRunApiTestSuite))
}

func (s *RecurringRunApiTestSuite) TearDownSuite() {
	if *runIntegrationTests {
		if !*isDevMode {
			s.cleanUp()
		}
	}
}

/** ======== the following are util functions ========= **/

func (s *RecurringRunApiTestSuite) cleanUp() {
	test.DeleteAllRuns(s.runClient, s.resourceNamespace, s.T())
	test.DeleteAllRecurringRuns(s.recurringRunClient, s.resourceNamespace, s.T())
	test.DeleteAllPipelines(s.pipelineClient, s.T())
	test.DeleteAllExperiments(s.experimentClient, s.resourceNamespace, s.T())
}

func defaultV2beta1RecurringRun(pipelineId, pipelineVersionId, experimentId string) *recurring_run_model.V2beta1RecurringRun {
	return &recurring_run_model.V2beta1RecurringRun{
		DisplayName:  "default-pipeline-name",
		Description:  "This is a default pipeline",
		ExperimentID: experimentId,
		PipelineVersionReference: &recurring_run_model.V2beta1PipelineVersionReference{
			PipelineID:        pipelineId,
			PipelineVersionID: pipelineVersionId,
		},
		MaxConcurrency: 10,
		NoCatchup:      false,
		Trigger: &recurring_run_model.V2beta1Trigger{
			PeriodicSchedule: &recurring_run_model.V2beta1PeriodicSchedule{
				StartTime:      strfmt.NewDateTime(),
				EndTime:        strfmt.NewDateTime(),
				IntervalSecond: 60,
			},
		},
		Mode: recurring_run_model.RecurringRunModeENABLE,
	}
}

type recurringRunOptions struct {
	pipelineId, pipelineVersionId, experimentId string
	periodic                                    bool
}

func recurringRunInThePastForTwoMinutes(options recurringRunOptions) *recurring_run_model.V2beta1RecurringRun {
	startTime := strfmt.DateTime(time.Unix(10*hour, 0))
	endTime := strfmt.DateTime(time.Unix(10*hour+2*minute, 0))

	recurringRun := defaultV2beta1RecurringRun(options.pipelineId, options.pipelineVersionId, options.experimentId)
	if options.periodic {
		recurringRun.Trigger = &recurring_run_model.V2beta1Trigger{
			PeriodicSchedule: &recurring_run_model.V2beta1PeriodicSchedule{
				StartTime:      startTime,
				EndTime:        endTime,
				IntervalSecond: 60, // Runs every 1 minute.
			},
		}
	} else {
		recurringRun.Trigger = &recurring_run_model.V2beta1Trigger{
			CronSchedule: &recurring_run_model.V2beta1CronSchedule{
				StartTime: startTime,
				EndTime:   endTime,
				Cron:      "0 * * * * ?", // Runs every 1 minute.
			},
		}
	}
	return recurringRun
}
