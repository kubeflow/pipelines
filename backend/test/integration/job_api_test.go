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
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/eapache/go-resiliency/retrier"
	"github.com/go-openapi/strfmt"
	"github.com/golang/glog"
	experimentparams "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/experiment_client/experiment_service"
	jobparams "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/job_client/job_service"
	"github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/job_model"
	uploadParams "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	runParams "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/run_model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	api_server "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v1"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

const (
	second = 1
	minute = 60 * second
	hour   = 60 * minute
)

type JobApiTestSuite struct {
	suite.Suite
	namespace            string
	resourceNamespace    string
	experimentClient     *api_server.ExperimentClient
	pipelineClient       *api_server.PipelineClient
	pipelineUploadClient *api_server.PipelineUploadClient
	runClient            *api_server.RunClient
	jobClient            *api_server.JobClient
	swfClient            client.SwfClientInterface
}

type JobResourceReferenceSorter []*job_model.APIResourceReference

func (r JobResourceReferenceSorter) Len() int           { return len(r) }
func (r JobResourceReferenceSorter) Less(i, j int) bool { return r[i].Name < r[j].Name }
func (r JobResourceReferenceSorter) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }

// Check the namespace have ML pipeline installed and ready
func (s *JobApiTestSuite) SetupTest() {
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
	var newJobClient func() (*api_server.JobClient, error)

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
		newJobClient = func() (*api_server.JobClient, error) {
			return api_server.NewKubeflowInClusterJobClient(s.namespace, *isDebugMode)
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
		newJobClient = func() (*api_server.JobClient, error) {
			return api_server.NewJobClient(clientConfig, *isDebugMode)
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
	s.jobClient, err = newJobClient()
	if err != nil {
		glog.Exitf("Failed to get job client. Error: %s", err.Error())
	}
	s.swfClient = client.NewScheduledWorkflowClientOrFatal(time.Second*30, util.ClientParameters{QPS: 5, Burst: 10})

	s.cleanUp()
}

func (s *JobApiTestSuite) TestJobApis() {
	t := s.T()

	/* ---------- Upload pipelines YAML ---------- */
	helloWorldPipeline, err := s.pipelineUploadClient.UploadFile("../resources/hello-world.yaml", uploadParams.NewUploadPipelineParams())
	assert.Nil(t, err)

	/* ---------- Upload pipeline version YAML ---------- */
	time.Sleep(1 * time.Second)
	helloWorldPipelineVersion, err := s.pipelineUploadClient.UploadPipelineVersion(
		"../resources/hello-world.yaml", &uploadParams.UploadPipelineVersionParams{
			Name:       util.StringPointer("hello-world-version"),
			Pipelineid: util.StringPointer(helloWorldPipeline.ID),
		})
	assert.Nil(t, err)

	/* ---------- Create a new hello world experiment ---------- */
	experiment := test.GetExperiment("hello world experiment", "", s.resourceNamespace)
	helloWorldExperiment, err := s.experimentClient.Create(&experimentparams.ExperimentServiceCreateExperimentV1Params{Body: experiment})
	assert.Nil(t, err)

	/* ---------- Create a new hello world job by specifying pipeline ID ---------- */
	createJobRequest := &jobparams.JobServiceCreateJobParams{Body: &job_model.APIJob{
		Name:        "hello world",
		Description: "this is hello world",
		ResourceReferences: []*job_model.APIResourceReference{
			{
				Key:          &job_model.APIResourceKey{Type: job_model.APIResourceTypeEXPERIMENT, ID: helloWorldExperiment.ID},
				Relationship: job_model.APIRelationshipOWNER,
			},
			{
				Key:          &job_model.APIResourceKey{Type: job_model.APIResourceTypePIPELINEVERSION, ID: helloWorldPipelineVersion.ID},
				Relationship: job_model.APIRelationshipCREATOR,
			},
		},
		MaxConcurrency: 10,
		Enabled:        true,
	}}
	helloWorldJob, err := s.jobClient.Create(createJobRequest)
	assert.Nil(t, err)
	s.checkHelloWorldJob(t, helloWorldJob, helloWorldExperiment.ID, helloWorldExperiment.Name, helloWorldPipelineVersion.ID, helloWorldPipelineVersion.Name)

	/* ---------- Get hello world job ---------- */
	helloWorldJob, err = s.jobClient.Get(&jobparams.JobServiceGetJobParams{ID: helloWorldJob.ID})
	assert.Nil(t, err)
	s.checkHelloWorldJob(t, helloWorldJob, helloWorldExperiment.ID, helloWorldExperiment.Name, helloWorldPipelineVersion.ID, helloWorldPipelineVersion.Name)

	/* ---------- Create a new argument parameter experiment ---------- */
	experiment = test.GetExperiment("argument parameter experiment", "", s.resourceNamespace)
	argParamsExperiment, err := s.experimentClient.Create(&experimentparams.ExperimentServiceCreateExperimentV1Params{Body: experiment})
	assert.Nil(t, err)

	/* ---------- Create a new argument parameter job by uploading workflow manifest ---------- */
	// Make sure the job is created at least 1 second later than the first one,
	// because sort by created_at has precision of 1 second.
	time.Sleep(1 * time.Second)
	argParamsBytes, err := ioutil.ReadFile("../resources/arguments-parameters.yaml")
	assert.Nil(t, err)
	argParamsBytes, err = yaml.ToJSON(argParamsBytes)
	assert.Nil(t, err)
	createJobRequest = &jobparams.JobServiceCreateJobParams{Body: &job_model.APIJob{
		Name:        "argument parameter",
		Description: "this is argument parameter",
		PipelineSpec: &job_model.APIPipelineSpec{
			WorkflowManifest: string(argParamsBytes),
			Parameters: []*job_model.APIParameter{
				{Name: "param1", Value: "goodbye"},
				{Name: "param2", Value: "world"},
			},
		},
		ResourceReferences: []*job_model.APIResourceReference{
			{
				Key:          &job_model.APIResourceKey{Type: job_model.APIResourceTypeEXPERIMENT, ID: argParamsExperiment.ID},
				Relationship: job_model.APIRelationshipOWNER,
			},
		},
		MaxConcurrency: 10,
		Enabled:        true,
	}}
	argParamsJob, err := s.jobClient.Create(createJobRequest)
	assert.Nil(t, err)
	s.checkArgParamsJob(t, argParamsJob, argParamsExperiment.ID, argParamsExperiment.Name)

	/* ---------- List all the jobs. Both jobs should be returned ---------- */
	jobs, totalSize, _, err := test.ListAllJobs(s.jobClient, s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 2, totalSize, "Incorrect total number of recurring runs in the namespace %v", s.resourceNamespace)
	assert.Equal(t, 2, len(jobs), "Incorrect total number of recurring runs in the namespace %v", s.resourceNamespace)

	/* ---------- List the jobs, paginated, sort by creation time ---------- */
	jobs, totalSize, nextPageToken, err := test.ListJobs(
		s.jobClient,
		&jobparams.JobServiceListJobsParams{
			PageSize: util.Int32Pointer(1),
			SortBy:   util.StringPointer("created_at"),
		},
		s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(jobs))
	assert.Equal(t, 2, totalSize)
	assert.Equal(t, "hello world", jobs[0].Name)
	jobs, totalSize, _, err = test.ListJobs(
		s.jobClient,
		&jobparams.JobServiceListJobsParams{
			PageSize:  util.Int32Pointer(1),
			PageToken: util.StringPointer(nextPageToken),
		},
		s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(jobs))
	assert.Equal(t, 2, totalSize)
	assert.Equal(t, "argument parameter", jobs[0].Name)

	/* ---------- List the jobs, paginated, sort by name ---------- */
	jobs, totalSize, nextPageToken, err = test.ListJobs(
		s.jobClient,
		&jobparams.JobServiceListJobsParams{
			PageSize: util.Int32Pointer(1),
			SortBy:   util.StringPointer("name"),
		},
		s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 2, totalSize)
	assert.Equal(t, 1, len(jobs))
	assert.Equal(t, "argument parameter", jobs[0].Name)
	jobs, totalSize, _, err = test.ListJobs(
		s.jobClient,
		&jobparams.JobServiceListJobsParams{
			PageSize:  util.Int32Pointer(1),
			SortBy:    util.StringPointer("name"),
			PageToken: util.StringPointer(nextPageToken),
		},
		s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 2, totalSize)
	assert.Equal(t, 1, len(jobs))
	assert.Equal(t, "hello world", jobs[0].Name)

	/* ---------- List the jobs, sort by unsupported field ---------- */
	jobs, _, _, err = test.ListJobs(
		s.jobClient,
		&jobparams.JobServiceListJobsParams{
			PageSize: util.Int32Pointer(2),
			SortBy:   util.StringPointer("unknown"),
		},
		s.resourceNamespace)
	assert.NotNil(t, err)
	assert.Equal(t, len(jobs), 0)

	/* ---------- List jobs for hello world experiment. One job should be returned ---------- */
	jobs, totalSize, _, err = s.jobClient.List(&jobparams.JobServiceListJobsParams{
		ResourceReferenceKeyType: util.StringPointer(string(run_model.APIResourceTypeEXPERIMENT)),
		ResourceReferenceKeyID:   util.StringPointer(helloWorldExperiment.ID),
	})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(jobs))
	assert.Equal(t, 1, totalSize)
	assert.Equal(t, "hello world", jobs[0].Name)

	/* ---------- List the jobs, filtered by created_at, only return the previous two jobs ---------- */
	time.Sleep(5 * time.Second) // Sleep for 5 seconds to make sure the previous jobs are created at a different timestamp
	filterTime := time.Now().Unix()
	time.Sleep(5 * time.Second)
	createJobRequestNew := &jobparams.JobServiceCreateJobParams{Body: &job_model.APIJob{
		Name:        "new hello world job",
		Description: "this is a new hello world",
		ResourceReferences: []*job_model.APIResourceReference{
			{
				Key:          &job_model.APIResourceKey{Type: job_model.APIResourceTypeEXPERIMENT, ID: helloWorldExperiment.ID},
				Relationship: job_model.APIRelationshipOWNER,
			},
			{
				Key:          &job_model.APIResourceKey{Type: job_model.APIResourceTypePIPELINEVERSION, ID: helloWorldPipelineVersion.ID},
				Relationship: job_model.APIRelationshipCREATOR,
			},
		},
		MaxConcurrency: 10,
		Enabled:        true,
	}}
	_, err = s.jobClient.Create(createJobRequestNew)
	assert.Nil(t, err)
	// Check total number of jobs to be 3
	jobs, totalSize, _, err = test.ListAllJobs(s.jobClient, s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 3, totalSize)
	assert.Equal(t, 3, len(jobs))
	// Check number of filtered jobs finished before filterTime to be 2
	jobs, totalSize, _, err = test.ListJobs(
		s.jobClient,
		&jobparams.JobServiceListJobsParams{
			Filter: util.StringPointer(`{"predicates": [{"key": "created_at", "op": 6, "string_value": "` + fmt.Sprint(filterTime) + `"}]}`),
		},
		s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(jobs))
	assert.Equal(t, 2, totalSize)

	// The scheduledWorkflow CRD would create the run and it synced to the DB by persistent agent.
	// This could take a few seconds to finish.

	/* ---------- Check run for hello world job ---------- */
	if err := retrier.New(retrier.ConstantBackoff(8, 5*time.Second), nil).Run(func() error {
		runs, totalSize, _, err := s.runClient.List(&runParams.RunServiceListRunsV1Params{
			ResourceReferenceKeyType: util.StringPointer(string(run_model.APIResourceTypeEXPERIMENT)),
			ResourceReferenceKeyID:   util.StringPointer(helloWorldExperiment.ID),
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
		return s.checkHelloWorldRun(helloWorldRun, helloWorldExperiment.ID, helloWorldExperiment.Name, helloWorldJob.ID, helloWorldJob.Name)
	}); err != nil {
		assert.Nil(t, err)
	}

	/* ---------- Check run for argument parameter job ---------- */
	if err := retrier.New(retrier.ConstantBackoff(8, 5*time.Second), nil).Run(func() error {
		runs, totalSize, _, err := s.runClient.List(&runParams.RunServiceListRunsV1Params{
			ResourceReferenceKeyType: util.StringPointer(string(run_model.APIResourceTypeEXPERIMENT)),
			ResourceReferenceKeyID:   util.StringPointer(argParamsExperiment.ID),
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
		return s.checkArgParamsRun(argParamsRun, argParamsExperiment.ID, argParamsExperiment.Name, argParamsJob.ID, argParamsJob.Name)
	}); err != nil {
		assert.Nil(t, err)
	}
}

func (s *JobApiTestSuite) TestJobApis_noCatchupOption() {
	t := s.T()

	/* ---------- Upload pipelines YAML ---------- */
	pipeline, err := s.pipelineUploadClient.UploadFile("../resources/hello-world.yaml", uploadParams.NewUploadPipelineParams())
	assert.Nil(t, err)

	/* ---------- Upload pipeline version YAML ---------- */
	time.Sleep(1 * time.Second)
	helloWorldPipelineVersion, err := s.pipelineUploadClient.UploadPipelineVersion(
		"../resources/hello-world.yaml", &uploadParams.UploadPipelineVersionParams{
			Name:       util.StringPointer("hello-world-version"),
			Pipelineid: util.StringPointer(pipeline.ID),
		})
	assert.Nil(t, err)

	/* ---------- Create a periodic job with start and end date in the past and catchup = true ---------- */
	experiment := test.GetExperiment("periodic catchup true", "", s.resourceNamespace)
	periodicCatchupTrueExperiment, err := s.experimentClient.Create(&experimentparams.ExperimentServiceCreateExperimentV1Params{Body: experiment})
	assert.Nil(t, err)

	job := jobInThePastForTwoMinutes(jobOptions{
		pipelineVersionId: helloWorldPipelineVersion.ID,
		experimentId:      periodicCatchupTrueExperiment.ID,
		periodic:          true,
	})
	job.Name = "periodic-catchup-true-"
	job.Description = "A job with NoCatchup=false will backfill each past interval when behind schedule."
	job.NoCatchup = false // This is the key difference.
	createJobRequest := &jobparams.JobServiceCreateJobParams{Body: job}
	_, err = s.jobClient.Create(createJobRequest)
	assert.Nil(t, err)

	/* -------- Create another periodic job with start and end date in the past but catchup = false ------ */
	experiment = test.GetExperiment("periodic catchup false", "", s.resourceNamespace)
	periodicCatchupFalseExperiment, err := s.experimentClient.Create(&experimentparams.ExperimentServiceCreateExperimentV1Params{Body: experiment})
	assert.Nil(t, err)

	job = jobInThePastForTwoMinutes(jobOptions{
		pipelineVersionId: helloWorldPipelineVersion.ID,
		experimentId:      periodicCatchupFalseExperiment.ID,
		periodic:          true,
	})
	job.Name = "periodic-catchup-false-"
	job.Description = "A job with NoCatchup=true only schedules the last interval when behind schedule."
	job.NoCatchup = true // This is the key difference.
	createJobRequest = &jobparams.JobServiceCreateJobParams{Body: job}
	_, err = s.jobClient.Create(createJobRequest)
	assert.Nil(t, err)

	/* ---------- Create a cron job with start and end date in the past and catchup = true ---------- */
	experiment = test.GetExperiment("cron catchup true", "", s.resourceNamespace)
	cronCatchupTrueExperiment, err := s.experimentClient.Create(&experimentparams.ExperimentServiceCreateExperimentV1Params{Body: experiment})
	assert.Nil(t, err)

	job = jobInThePastForTwoMinutes(jobOptions{
		pipelineVersionId: helloWorldPipelineVersion.ID,
		experimentId:      cronCatchupTrueExperiment.ID,
		periodic:          false,
	})
	job.Name = "cron-catchup-true-"
	job.Description = "A job with NoCatchup=false will backfill each past interval when behind schedule."
	job.NoCatchup = false // This is the key difference.
	createJobRequest = &jobparams.JobServiceCreateJobParams{Body: job}
	_, err = s.jobClient.Create(createJobRequest)
	assert.Nil(t, err)

	/* -------- Create another cron job with start and end date in the past but catchup = false ------ */
	experiment = test.GetExperiment("cron catchup false", "", s.resourceNamespace)
	cronCatchupFalseExperiment, err := s.experimentClient.Create(&experimentparams.ExperimentServiceCreateExperimentV1Params{Body: experiment})
	assert.Nil(t, err)

	job = jobInThePastForTwoMinutes(jobOptions{
		pipelineVersionId: helloWorldPipelineVersion.ID,
		experimentId:      cronCatchupFalseExperiment.ID,
		periodic:          false,
	})
	job.Name = "cron-catchup-false-"
	job.Description = "A job with NoCatchup=true only schedules the last interval when behind schedule."
	job.NoCatchup = true // This is the key difference.
	createJobRequest = &jobparams.JobServiceCreateJobParams{Body: job}
	_, err = s.jobClient.Create(createJobRequest)
	assert.Nil(t, err)

	// The scheduledWorkflow CRD would create the run and it is synced to the DB by persistent agent.
	// This could take a few seconds to finish.

	/* ---------- Assert number of runs when catchup = true ---------- */
	if err := retrier.New(retrier.ConstantBackoff(8, 5*time.Second), nil).Run(func() error {
		_, runsWhenCatchupTrue, _, err := s.runClient.List(&runParams.RunServiceListRunsV1Params{
			ResourceReferenceKeyType: util.StringPointer(string(run_model.APIResourceTypeEXPERIMENT)),
			ResourceReferenceKeyID:   util.StringPointer(periodicCatchupTrueExperiment.ID),
		})
		if err != nil {
			return err
		}
		if runsWhenCatchupTrue != 2 {
			return fmt.Errorf("expected runsWhenCatchupTrue with periodic schedule to be 2, got: %v", runsWhenCatchupTrue)
		}

		_, runsWhenCatchupTrue, _, err = s.runClient.List(&runParams.RunServiceListRunsV1Params{
			ResourceReferenceKeyType: util.StringPointer(string(run_model.APIResourceTypeEXPERIMENT)),
			ResourceReferenceKeyID:   util.StringPointer(cronCatchupTrueExperiment.ID),
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
		_, runsWhenCatchupFalse, _, err := s.runClient.List(&runParams.RunServiceListRunsV1Params{
			ResourceReferenceKeyType: util.StringPointer(string(run_model.APIResourceTypeEXPERIMENT)),
			ResourceReferenceKeyID:   util.StringPointer(periodicCatchupFalseExperiment.ID),
		})
		if err != nil {
			return err
		}
		if runsWhenCatchupFalse != 1 {
			return fmt.Errorf("expected runsWhenCatchupFalse with periodic schedule to be 1, got: %v", runsWhenCatchupFalse)
		}

		_, runsWhenCatchupFalse, _, err = s.runClient.List(&runParams.RunServiceListRunsV1Params{
			ResourceReferenceKeyType: util.StringPointer(string(run_model.APIResourceTypeEXPERIMENT)),
			ResourceReferenceKeyID:   util.StringPointer(cronCatchupFalseExperiment.ID),
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

func (s *JobApiTestSuite) checkHelloWorldJob(t *testing.T, job *job_model.APIJob, experimentID string, experimentName string, pipelineVersionId string, pipelineVersionName string) {
	// Check workflow manifest is not empty
	assert.Contains(t, job.PipelineSpec.WorkflowManifest, "whalesay")

	expectedJob := &job_model.APIJob{
		ID:             job.ID,
		Name:           "hello world",
		Description:    "this is hello world",
		ServiceAccount: test.GetDefaultPipelineRunnerServiceAccount(*isKubeflowMode),
		PipelineSpec: &job_model.APIPipelineSpec{
			PipelineID:       job.PipelineSpec.PipelineID,
			PipelineName:     job.PipelineSpec.PipelineName,
			WorkflowManifest: job.PipelineSpec.WorkflowManifest,
		},
		ResourceReferences: []*job_model.APIResourceReference{
			{
				Key:  &job_model.APIResourceKey{Type: job_model.APIResourceTypeEXPERIMENT, ID: experimentID},
				Name: experimentName, Relationship: job_model.APIRelationshipOWNER,
			},
			{
				Key:  &job_model.APIResourceKey{Type: job_model.APIResourceTypePIPELINEVERSION, ID: pipelineVersionId},
				Name: pipelineVersionName, Relationship: job_model.APIRelationshipCREATOR,
			},
		},
		MaxConcurrency: 10,
		Enabled:        true,
		CreatedAt:      job.CreatedAt,
		UpdatedAt:      job.UpdatedAt,
		Status:         job.Status,
	}
	assert.True(t, test.VerifyJobResourceReferences(job.ResourceReferences, expectedJob.ResourceReferences))
	expectedJob.ResourceReferences = job.ResourceReferences

	assert.Equal(t, expectedJob, job)
}

func (s *JobApiTestSuite) checkArgParamsJob(t *testing.T, job *job_model.APIJob, experimentID string, experimentName string) {
	// Check runtime workflow manifest is not empty
	assert.Contains(t, job.PipelineSpec.WorkflowManifest, "arguments-parameters-", "Job: %v", job.PipelineSpec)
	expectedJob := &job_model.APIJob{
		ID:             job.ID,
		Name:           "argument parameter",
		Description:    "this is argument parameter",
		ServiceAccount: test.GetDefaultPipelineRunnerServiceAccount(*isKubeflowMode),
		PipelineSpec: &job_model.APIPipelineSpec{
			PipelineID:       job.PipelineSpec.PipelineID,
			PipelineName:     job.PipelineSpec.PipelineName,
			WorkflowManifest: job.PipelineSpec.WorkflowManifest,
			Parameters: []*job_model.APIParameter{
				{Name: "param1", Value: "goodbye"},
				{Name: "param2", Value: "world"},
			},
		},
		ResourceReferences: []*job_model.APIResourceReference{
			{
				Key:  &job_model.APIResourceKey{Type: job_model.APIResourceTypeEXPERIMENT, ID: experimentID},
				Name: experimentName, Relationship: job_model.APIRelationshipOWNER,
			},
		},
		MaxConcurrency: 10,
		Enabled:        true,
		CreatedAt:      job.CreatedAt,
		UpdatedAt:      job.UpdatedAt,
		Status:         job.Status,
	}
	assert.True(t, test.VerifyJobResourceReferences(job.ResourceReferences, expectedJob.ResourceReferences))
	expectedJob.ResourceReferences = job.ResourceReferences
	assert.Equal(t, expectedJob, job)
}

func (s *JobApiTestSuite) TestJobApis_SwfNotFound() {
	t := s.T()

	/* ---------- Upload pipelines YAML ---------- */
	pipeline, err := s.pipelineUploadClient.UploadFile("../resources/hello-world.yaml", uploadParams.NewUploadPipelineParams())
	require.Nil(t, err)

	/* ---------- Create a new hello world job by specifying pipeline ID ---------- */
	createJobRequest := &jobparams.JobServiceCreateJobParams{Body: &job_model.APIJob{
		Name: "test-swf-not-found",
		PipelineSpec: &job_model.APIPipelineSpec{
			PipelineID: pipeline.ID,
		},
		MaxConcurrency: 10,
		Enabled:        false,
	}}
	// In multi-user mode, jobs must be associated with an experiment.
	if *isKubeflowMode {
		experiment := test.GetExperiment("test-swf-not-found experiment", "", s.resourceNamespace)
		swfNotFoundExperiment, err := s.experimentClient.Create(&experimentparams.ExperimentServiceCreateExperimentV1Params{Body: experiment})
		assert.Nil(t, err)

		createJobRequest.Body.ResourceReferences = []*job_model.APIResourceReference{
			{Key: &job_model.APIResourceKey{Type: job_model.APIResourceTypeEXPERIMENT, ID: swfNotFoundExperiment.ID},
				Relationship: job_model.APIRelationshipOWNER,
			},
		}
	}
	job, err := s.jobClient.Create(createJobRequest)
	require.Nil(t, err)

	// Delete all ScheduledWorkflow custom resources to simulate the situation
	// that after reinstalling KFP with managed storage, only KFP DB is kept,
	// but all KFP custom resources are gone.
	swfNamespace := s.namespace
	if s.resourceNamespace != "" {
		swfNamespace = s.resourceNamespace
	}
	err = s.swfClient.ScheduledWorkflow(swfNamespace).DeleteCollection(context.Background(), &v1.DeleteOptions{}, v1.ListOptions{})
	require.Nil(t, err)

	err = s.jobClient.Delete(&jobparams.JobServiceDeleteJobParams{ID: job.ID})
	require.Nil(t, err)

	/* ---------- Get job ---------- */
	_, err = s.jobClient.Get(&jobparams.JobServiceGetJobParams{ID: job.ID})
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "not found")
}

func equal(expected, actual interface{}) bool {
	if expected == nil || actual == nil {
		return expected == actual
	}

	exp, ok := expected.([]byte)
	if !ok {
		return reflect.DeepEqual(expected, actual)
	}

	act, ok := actual.([]byte)
	if !ok {
		return false
	}
	if exp == nil || act == nil {
		return exp == nil && act == nil
	}
	return bytes.Equal(exp, act)
}

func (s *JobApiTestSuite) checkHelloWorldRun(run *run_model.APIRun, experimentID string, experimentName string, jobID string, jobName string) error {
	// Check workflow manifest is not empty
	if !strings.Contains(run.PipelineSpec.WorkflowManifest, "whalesay") {
		return fmt.Errorf("expected: %+v got: %+v", "whalesay", run.PipelineSpec.WorkflowManifest)
	}

	if !strings.Contains(run.Name, "helloworld") {
		return fmt.Errorf("expected: %+v got: %+v", "helloworld", run.Name)
	}

	// Check runtime workflow manifest is not empty
	resourceReferences := []*run_model.APIResourceReference{
		{
			Key:  &run_model.APIResourceKey{Type: run_model.APIResourceTypeEXPERIMENT, ID: experimentID},
			Name: experimentName, Relationship: run_model.APIRelationshipOWNER,
		},
		{
			Key:  &run_model.APIResourceKey{Type: run_model.APIResourceTypeJOB, ID: jobID},
			Name: jobName, Relationship: run_model.APIRelationshipCREATOR,
		},
	}
	if !test.VerifyRunResourceReferences(run.ResourceReferences, resourceReferences) {
		return fmt.Errorf("expected: %+v got: %+v", resourceReferences, run.ResourceReferences)
	}

	return nil
}

func (s *JobApiTestSuite) checkArgParamsRun(run *run_model.APIRun, experimentID string, experimentName string, jobID string, jobName string) error {
	if !strings.Contains(run.Name, "argumentparameter") {
		return fmt.Errorf("expected: %+v got: %+v", "argumentparameter", run.Name)
	}
	// Check runtime workflow manifest is not empty
	resourceReferences := []*run_model.APIResourceReference{
		{
			Key:  &run_model.APIResourceKey{Type: run_model.APIResourceTypeEXPERIMENT, ID: experimentID},
			Name: experimentName, Relationship: run_model.APIRelationshipOWNER,
		},
		{
			Key:  &run_model.APIResourceKey{Type: run_model.APIResourceTypeJOB, ID: jobID},
			Name: jobName, Relationship: run_model.APIRelationshipCREATOR,
		},
	}
	if !test.VerifyRunResourceReferences(run.ResourceReferences, resourceReferences) {
		return fmt.Errorf("expected: %+v got: %+v", resourceReferences, run.ResourceReferences)
	}
	return nil
}

func TestJobApi(t *testing.T) {
	suite.Run(t, new(JobApiTestSuite))
}

func (s *JobApiTestSuite) TearDownSuite() {
	if *runIntegrationTests {
		if !*isDevMode {
			s.cleanUp()
		}
	}
}

/** ======== the following are util functions ========= **/

func (s *JobApiTestSuite) cleanUp() {
	test.DeleteAllRuns(s.runClient, s.resourceNamespace, s.T())
	test.DeleteAllJobs(s.jobClient, s.resourceNamespace, s.T())
	test.DeleteAllPipelines(s.pipelineClient, s.T())
	test.DeleteAllExperiments(s.experimentClient, s.resourceNamespace, s.T())
}

func defaultApiJob(pipelineVersionId, experimentId string) *job_model.APIJob {
	return &job_model.APIJob{
		Name:        "default-pipeline-name",
		Description: "This is a default pipeline",
		ResourceReferences: []*job_model.APIResourceReference{
			{
				Key:          &job_model.APIResourceKey{Type: job_model.APIResourceTypeEXPERIMENT, ID: experimentId},
				Relationship: job_model.APIRelationshipOWNER,
			},
			{
				Key:          &job_model.APIResourceKey{Type: job_model.APIResourceTypePIPELINEVERSION, ID: pipelineVersionId},
				Relationship: job_model.APIRelationshipCREATOR,
			},
		},
		MaxConcurrency: 10,
		NoCatchup:      false,
		Trigger: &job_model.APITrigger{
			PeriodicSchedule: &job_model.APIPeriodicSchedule{
				StartTime:      strfmt.NewDateTime(),
				EndTime:        strfmt.NewDateTime(),
				IntervalSecond: 60,
			},
		},
		Enabled: true,
	}
}

type jobOptions struct {
	pipelineVersionId, experimentId string
	periodic                        bool
}

func jobInThePastForTwoMinutes(options jobOptions) *job_model.APIJob {
	startTime := strfmt.DateTime(time.Unix(10*hour, 0))
	endTime := strfmt.DateTime(time.Unix(10*hour+2*minute, 0))

	job := defaultApiJob(options.pipelineVersionId, options.experimentId)
	if options.periodic {
		job.Trigger = &job_model.APITrigger{
			PeriodicSchedule: &job_model.APIPeriodicSchedule{
				StartTime:      startTime,
				EndTime:        endTime,
				IntervalSecond: 60, // Runs every 1 minute.
			},
		}
	} else {
		job.Trigger = &job_model.APITrigger{
			CronSchedule: &job_model.APICronSchedule{
				StartTime: startTime,
				EndTime:   endTime,
				Cron:      "0 * * * * ?", // Runs every 1 minute.
			},
		}
	}
	return job
}
