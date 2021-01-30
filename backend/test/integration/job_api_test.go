package integration

import (
	"io/ioutil"
	"sort"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/test"

	"github.com/go-openapi/strfmt"
	"github.com/golang/glog"
	experimentparams "github.com/kubeflow/pipelines/backend/api/go_http_client/experiment_client/experiment_service"
	"github.com/kubeflow/pipelines/backend/api/go_http_client/experiment_model"
	jobparams "github.com/kubeflow/pipelines/backend/api/go_http_client/job_client/job_service"
	"github.com/kubeflow/pipelines/backend/api/go_http_client/job_model"
	uploadParams "github.com/kubeflow/pipelines/backend/api/go_http_client/pipeline_upload_client/pipeline_upload_service"
	runParams "github.com/kubeflow/pipelines/backend/api/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/api/go_http_client/run_model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/common/client/api_server"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
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
	conn                 *grpc.ClientConn
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
		err := test.WaitForReady(*namespace, *initializeTimeout)
		if err != nil {
			glog.Exitf("Failed to initialize test. Error: %s", err.Error())
		}
	}
	s.namespace = *namespace
	clientConfig := test.GetClientConfig(*namespace)
	var err error
	s.experimentClient, err = api_server.NewExperimentClient(clientConfig, false)
	if err != nil {
		glog.Exitf("Failed to get pipeline upload client. Error: %s", err.Error())
	}
	s.pipelineUploadClient, err = api_server.NewPipelineUploadClient(clientConfig, false)
	if err != nil {
		glog.Exitf("Failed to get pipeline upload client. Error: %s", err.Error())
	}
	s.pipelineClient, err = api_server.NewPipelineClient(clientConfig, false)
	if err != nil {
		glog.Exitf("Failed to get pipeline client. Error: %s", err.Error())
	}
	s.runClient, err = api_server.NewRunClient(clientConfig, false)
	if err != nil {
		glog.Exitf("Failed to get run client. Error: %s", err.Error())
	}
	s.jobClient, err = api_server.NewJobClient(clientConfig, false)
	if err != nil {
		glog.Exitf("Failed to get job client. Error: %s", err.Error())
	}
	s.swfClient = client.NewScheduledWorkflowClientOrFatal(time.Second * 30)

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
	experiment := &experiment_model.APIExperiment{Name: "hello world experiment"}
	helloWorldExperiment, err := s.experimentClient.Create(&experimentparams.CreateExperimentParams{Body: experiment})
	assert.Nil(t, err)

	/* ---------- Create a new hello world job by specifying pipeline ID ---------- */
	createJobRequest := &jobparams.CreateJobParams{Body: &job_model.APIJob{
		Name:        "hello world",
		Description: "this is hello world",
		ResourceReferences: []*job_model.APIResourceReference{
			{Key: &job_model.APIResourceKey{Type: job_model.APIResourceTypeEXPERIMENT, ID: helloWorldExperiment.ID},
				Relationship: job_model.APIRelationshipOWNER},
			{Key: &job_model.APIResourceKey{Type: job_model.APIResourceTypePIPELINEVERSION, ID: helloWorldPipelineVersion.ID},
				Relationship: job_model.APIRelationshipCREATOR},
		},
		MaxConcurrency: 10,
		Enabled:        true,
	}}
	helloWorldJob, err := s.jobClient.Create(createJobRequest)
	assert.Nil(t, err)
	s.checkHelloWorldJob(t, helloWorldJob, helloWorldExperiment.ID, helloWorldExperiment.Name, helloWorldPipelineVersion.ID, helloWorldPipelineVersion.Name)

	/* ---------- Get hello world job ---------- */
	helloWorldJob, err = s.jobClient.Get(&jobparams.GetJobParams{ID: helloWorldJob.ID})
	assert.Nil(t, err)
	s.checkHelloWorldJob(t, helloWorldJob, helloWorldExperiment.ID, helloWorldExperiment.Name, helloWorldPipelineVersion.ID, helloWorldPipelineVersion.Name)

	/* ---------- Create a new argument parameter experiment ---------- */
	experiment = &experiment_model.APIExperiment{Name: "argument parameter experiment"}
	argParamsExperiment, err := s.experimentClient.Create(&experimentparams.CreateExperimentParams{Body: experiment})
	assert.Nil(t, err)

	/* ---------- Create a new argument parameter job by uploading workflow manifest ---------- */
	// Make sure the job is created at least 1 second later than the first one,
	// because sort by created_at has precision of 1 second.
	time.Sleep(1 * time.Second)
	argParamsBytes, err := ioutil.ReadFile("../resources/arguments-parameters.yaml")
	assert.Nil(t, err)
	argParamsBytes, err = yaml.ToJSON(argParamsBytes)
	assert.Nil(t, err)
	createJobRequest = &jobparams.CreateJobParams{Body: &job_model.APIJob{
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
			{Key: &job_model.APIResourceKey{Type: job_model.APIResourceTypeEXPERIMENT, ID: argParamsExperiment.ID},
				Relationship: job_model.APIRelationshipOWNER},
		},
		MaxConcurrency: 10,
		Enabled:        true,
	}}
	argParamsJob, err := s.jobClient.Create(createJobRequest)
	assert.Nil(t, err)
	s.checkArgParamsJob(t, argParamsJob, argParamsExperiment.ID, argParamsExperiment.Name)

	/* ---------- List all the jobs. Both jobs should be returned ---------- */
	jobs, totalSize, _, err := s.jobClient.List(&jobparams.ListJobsParams{})
	assert.Nil(t, err)
	assert.Equal(t, 2, totalSize)
	assert.Equal(t, 2, len(jobs))

	/* ---------- List the jobs, paginated, sort by creation time ---------- */
	jobs, totalSize, nextPageToken, err := s.jobClient.List(
		&jobparams.ListJobsParams{PageSize: util.Int32Pointer(1), SortBy: util.StringPointer("created_at")})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(jobs))
	assert.Equal(t, 2, totalSize)
	assert.Equal(t, "hello world", jobs[0].Name)
	jobs, totalSize, _, err = s.jobClient.List(&jobparams.ListJobsParams{
		PageSize: util.Int32Pointer(1), PageToken: util.StringPointer(nextPageToken)})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(jobs))
	assert.Equal(t, 2, totalSize)
	assert.Equal(t, "argument parameter", jobs[0].Name)

	/* ---------- List the jobs, paginated, sort by name ---------- */
	jobs, totalSize, nextPageToken, err = s.jobClient.List(&jobparams.ListJobsParams{
		PageSize: util.Int32Pointer(1), SortBy: util.StringPointer("name")})
	assert.Nil(t, err)
	assert.Equal(t, 2, totalSize)
	assert.Equal(t, 1, len(jobs))
	assert.Equal(t, "argument parameter", jobs[0].Name)
	jobs, totalSize, _, err = s.jobClient.List(&jobparams.ListJobsParams{
		PageSize: util.Int32Pointer(1), SortBy: util.StringPointer("name"), PageToken: util.StringPointer(nextPageToken)})
	assert.Nil(t, err)
	assert.Equal(t, 2, totalSize)
	assert.Equal(t, 1, len(jobs))
	assert.Equal(t, "hello world", jobs[0].Name)

	/* ---------- List the jobs, sort by unsupported field ---------- */
	jobs, _, _, err = s.jobClient.List(&jobparams.ListJobsParams{
		PageSize: util.Int32Pointer(2), SortBy: util.StringPointer("unknown")})
	assert.NotNil(t, err)

	/* ---------- List jobs for hello world experiment. One job should be returned ---------- */
	jobs, totalSize, _, err = s.jobClient.List(&jobparams.ListJobsParams{
		ResourceReferenceKeyType: util.StringPointer(string(run_model.APIResourceTypeEXPERIMENT)),
		ResourceReferenceKeyID:   util.StringPointer(helloWorldExperiment.ID)})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(jobs))
	assert.Equal(t, 1, totalSize)
	assert.Equal(t, "hello world", jobs[0].Name)

	// The scheduledWorkflow CRD would create the run and it synced to the DB by persistent agent.
	// This could take a few seconds to finish.
	// TODO: Retry list run every 5 seconds instead of sleeping for 40 seconds.
	time.Sleep(40 * time.Second)

	/* ---------- Check run for hello world job ---------- */
	runs, totalSize, _, err := s.runClient.List(&runParams.ListRunsParams{
		ResourceReferenceKeyType: util.StringPointer(string(run_model.APIResourceTypeEXPERIMENT)),
		ResourceReferenceKeyID:   util.StringPointer(helloWorldExperiment.ID)})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(runs))
	assert.Equal(t, 1, totalSize)
	helloWorldRun := runs[0]
	s.checkHelloWorldRun(t, helloWorldRun, helloWorldExperiment.ID, helloWorldExperiment.Name, helloWorldJob.ID, helloWorldJob.Name)

	/* ---------- Check run for argument parameter job ---------- */
	runs, totalSize, _, err = s.runClient.List(&runParams.ListRunsParams{
		ResourceReferenceKeyType: util.StringPointer(string(run_model.APIResourceTypeEXPERIMENT)),
		ResourceReferenceKeyID:   util.StringPointer(argParamsExperiment.ID)})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(runs))
	assert.Equal(t, 1, totalSize)
	argParamsRun := runs[0]
	s.checkArgParamsRun(t, argParamsRun, argParamsExperiment.ID, argParamsExperiment.Name, argParamsJob.ID, argParamsJob.Name)
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
	experiment := &experiment_model.APIExperiment{Name: "periodic catchup true"}
	periodicCatchupTrueExperiment, err := s.experimentClient.Create(&experimentparams.CreateExperimentParams{Body: experiment})
	assert.Nil(t, err)

	job := jobInThePastForTwoMinutes(jobOptions{
		pipelineVersionId: helloWorldPipelineVersion.ID,
		experimentId:      periodicCatchupTrueExperiment.ID,
		periodic:          true,
	})
	job.Name = "periodic-catchup-true-"
	job.Description = "A job with NoCatchup=false will backfill each past interval when behind schedule."
	job.NoCatchup = false // This is the key difference.
	createJobRequest := &jobparams.CreateJobParams{Body: job}
	_, err = s.jobClient.Create(createJobRequest)
	assert.Nil(t, err)

	/* -------- Create another periodic job with start and end date in the past but catchup = false ------ */
	experiment = &experiment_model.APIExperiment{Name: "periodic catchup false"}
	periodicCatchupFalseExperiment, err := s.experimentClient.Create(&experimentparams.CreateExperimentParams{Body: experiment})
	assert.Nil(t, err)

	job = jobInThePastForTwoMinutes(jobOptions{
		pipelineVersionId: helloWorldPipelineVersion.ID,
		experimentId:      periodicCatchupFalseExperiment.ID,
		periodic:          true,
	})
	job.Name = "periodic-catchup-false-"
	job.Description = "A job with NoCatchup=true only schedules the last interval when behind schedule."
	job.NoCatchup = true // This is the key difference.
	createJobRequest = &jobparams.CreateJobParams{Body: job}
	_, err = s.jobClient.Create(createJobRequest)
	assert.Nil(t, err)

	/* ---------- Create a cron job with start and end date in the past and catchup = true ---------- */
	experiment = &experiment_model.APIExperiment{Name: "cron catchup true"}
	cronCatchupTrueExperiment, err := s.experimentClient.Create(&experimentparams.CreateExperimentParams{Body: experiment})
	assert.Nil(t, err)

	job = jobInThePastForTwoMinutes(jobOptions{
		pipelineVersionId: helloWorldPipelineVersion.ID,
		experimentId:      cronCatchupTrueExperiment.ID,
		periodic:          false,
	})
	job.Name = "cron-catchup-true-"
	job.Description = "A job with NoCatchup=false will backfill each past interval when behind schedule."
	job.NoCatchup = false // This is the key difference.
	createJobRequest = &jobparams.CreateJobParams{Body: job}
	_, err = s.jobClient.Create(createJobRequest)
	assert.Nil(t, err)

	/* -------- Create another cron job with start and end date in the past but catchup = false ------ */
	experiment = &experiment_model.APIExperiment{Name: "cron catchup false"}
	cronCatchupFalseExperiment, err := s.experimentClient.Create(&experimentparams.CreateExperimentParams{Body: experiment})
	assert.Nil(t, err)

	job = jobInThePastForTwoMinutes(jobOptions{
		pipelineVersionId: helloWorldPipelineVersion.ID,
		experimentId:      cronCatchupFalseExperiment.ID,
		periodic:          false,
	})
	job.Name = "cron-catchup-false-"
	job.Description = "A job with NoCatchup=true only schedules the last interval when behind schedule."
	job.NoCatchup = true // This is the key difference.
	createJobRequest = &jobparams.CreateJobParams{Body: job}
	_, err = s.jobClient.Create(createJobRequest)
	assert.Nil(t, err)

	// The scheduledWorkflow CRD would create the run and it synced to the DB by persistent agent.
	// This could take a few seconds to finish.
	// TODO: Retry list run every 5 seconds instead of sleeping for 40 seconds.
	time.Sleep(40 * time.Second)

	/* ---------- Assert number of runs when catchup = true ---------- */
	_, runsWhenCatchupTrue, _, err := s.runClient.List(&runParams.ListRunsParams{
		ResourceReferenceKeyType: util.StringPointer(string(run_model.APIResourceTypeEXPERIMENT)),
		ResourceReferenceKeyID:   util.StringPointer(periodicCatchupTrueExperiment.ID)})
	assert.Nil(t, err)
	assert.Equal(t, 2, runsWhenCatchupTrue)
	_, runsWhenCatchupTrue, _, err = s.runClient.List(&runParams.ListRunsParams{
		ResourceReferenceKeyType: util.StringPointer(string(run_model.APIResourceTypeEXPERIMENT)),
		ResourceReferenceKeyID:   util.StringPointer(cronCatchupTrueExperiment.ID)})
	assert.Equal(t, 2, runsWhenCatchupTrue)

	/* ---------- Assert number of runs when catchup = false ---------- */
	_, runsWhenCatchupFalse, _, err := s.runClient.List(&runParams.ListRunsParams{
		ResourceReferenceKeyType: util.StringPointer(string(run_model.APIResourceTypeEXPERIMENT)),
		ResourceReferenceKeyID:   util.StringPointer(periodicCatchupFalseExperiment.ID)})
	assert.Nil(t, err)
	assert.Equal(t, 1, runsWhenCatchupFalse)
	_, runsWhenCatchupFalse, _, err = s.runClient.List(&runParams.ListRunsParams{
		ResourceReferenceKeyType: util.StringPointer(string(run_model.APIResourceTypeEXPERIMENT)),
		ResourceReferenceKeyID:   util.StringPointer(cronCatchupFalseExperiment.ID)})
	assert.Nil(t, err)
	assert.Equal(t, 1, runsWhenCatchupFalse)
}

func (s *JobApiTestSuite) checkHelloWorldJob(t *testing.T, job *job_model.APIJob, experimentID string, experimentName string, pipelineVersionId string, pipelineVersionName string) {
	// Check workflow manifest is not empty
	assert.Contains(t, job.PipelineSpec.WorkflowManifest, "whalesay")

	expectedJob := &job_model.APIJob{
		ID:             job.ID,
		Name:           "hello world",
		Description:    "this is hello world",
		ServiceAccount: "pipeline-runner",
		PipelineSpec: &job_model.APIPipelineSpec{
			WorkflowManifest: job.PipelineSpec.WorkflowManifest,
		},
		ResourceReferences: []*job_model.APIResourceReference{
			{Key: &job_model.APIResourceKey{Type: job_model.APIResourceTypeEXPERIMENT, ID: experimentID},
				Name: experimentName, Relationship: job_model.APIRelationshipOWNER,
			},
			{Key: &job_model.APIResourceKey{Type: job_model.APIResourceTypePIPELINEVERSION, ID: pipelineVersionId},
				Name: pipelineVersionName, Relationship: job_model.APIRelationshipCREATOR},
		},
		MaxConcurrency: 10,
		Enabled:        true,
		CreatedAt:      job.CreatedAt,
		UpdatedAt:      job.UpdatedAt,
		Status:         job.Status,
		Trigger:        &job_model.APITrigger{},
	}

	// Need to sort resource references before equality check as the order is non-deterministic
	sort.Sort(JobResourceReferenceSorter(job.ResourceReferences))
	sort.Sort(JobResourceReferenceSorter(expectedJob.ResourceReferences))
	assert.Equal(t, expectedJob, job)
}

func (s *JobApiTestSuite) checkArgParamsJob(t *testing.T, job *job_model.APIJob, experimentID string, experimentName string) {
	argParamsBytes, err := ioutil.ReadFile("../resources/arguments-parameters.yaml")
	assert.Nil(t, err)
	argParamsBytes, err = yaml.ToJSON(argParamsBytes)
	assert.Nil(t, err)
	// Check runtime workflow manifest is not empty
	assert.Contains(t, job.PipelineSpec.WorkflowManifest, "arguments-parameters-")
	expectedJob := &job_model.APIJob{
		ID:             job.ID,
		Name:           "argument parameter",
		Description:    "this is argument parameter",
		ServiceAccount: "pipeline-runner",
		PipelineSpec: &job_model.APIPipelineSpec{
			WorkflowManifest: job.PipelineSpec.WorkflowManifest,
			Parameters: []*job_model.APIParameter{
				{Name: "param1", Value: "goodbye"},
				{Name: "param2", Value: "world"},
			},
		},
		ResourceReferences: []*job_model.APIResourceReference{
			{Key: &job_model.APIResourceKey{Type: job_model.APIResourceTypeEXPERIMENT, ID: experimentID},
				Name: experimentName, Relationship: job_model.APIRelationshipOWNER,
			},
		},
		MaxConcurrency: 10,
		Enabled:        true,
		CreatedAt:      job.CreatedAt,
		UpdatedAt:      job.UpdatedAt,
		Status:         job.Status,
		Trigger:        &job_model.APITrigger{},
	}

	assert.Equal(t, expectedJob, job)
}

func (s *JobApiTestSuite) TestJobApis_SwfNotFound() {
	t := s.T()

	/* ---------- Upload pipelines YAML ---------- */
	pipeline, err := s.pipelineUploadClient.UploadFile("../resources/hello-world.yaml", uploadParams.NewUploadPipelineParams())
	require.Nil(t, err)

	/* ---------- Create a new hello world job by specifying pipeline ID ---------- */
	createJobRequest := &jobparams.CreateJobParams{Body: &job_model.APIJob{
		Name: "test-swf-not-found",
		PipelineSpec: &job_model.APIPipelineSpec{
			PipelineID: pipeline.ID,
		},
		MaxConcurrency: 10,
		Enabled:        false,
	}}
	job, err := s.jobClient.Create(createJobRequest)
	require.Nil(t, err)

	// Delete all ScheduledWorkflow custom resources to simulate the situation
	// that after reinstalling KFP with managed storage, only KFP DB is kept,
	// but all KFP custom resources are gone.
	err = s.swfClient.ScheduledWorkflow(s.namespace).DeleteCollection(&v1.DeleteOptions{}, v1.ListOptions{})
	require.Nil(t, err)

	err = s.jobClient.Delete(&jobparams.DeleteJobParams{ID: job.ID})
	require.Nil(t, err)

	/* ---------- Get job ---------- */
	_, err = s.jobClient.Get(&jobparams.GetJobParams{ID: job.ID})
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "not found")
}

func (s *JobApiTestSuite) checkHelloWorldRun(t *testing.T, run *run_model.APIRun, experimentID string, experimentName string, jobID string, jobName string) {
	// Check workflow manifest is not empty
	assert.Contains(t, run.PipelineSpec.WorkflowManifest, "whalesay")
	assert.Contains(t, run.Name, "helloworld")
	// Check runtime workflow manifest is not empty
	resourceReferences := []*run_model.APIResourceReference{
		{Key: &run_model.APIResourceKey{Type: run_model.APIResourceTypeEXPERIMENT, ID: experimentID},
			Name: experimentName, Relationship: run_model.APIRelationshipOWNER,
		},
		{Key: &run_model.APIResourceKey{Type: run_model.APIResourceTypeJOB, ID: jobID},
			Name: jobName, Relationship: run_model.APIRelationshipCREATOR,
		},
	}
	assert.Equal(t, resourceReferences, run.ResourceReferences)
}

func (s *JobApiTestSuite) checkArgParamsRun(t *testing.T, run *run_model.APIRun, experimentID string, experimentName string, jobID string, jobName string) {
	assert.Contains(t, run.Name, "argumentparameter")
	// Check runtime workflow manifest is not empty
	resourceReferences := []*run_model.APIResourceReference{
		{Key: &run_model.APIResourceKey{Type: run_model.APIResourceTypeEXPERIMENT, ID: experimentID},
			Name: experimentName, Relationship: run_model.APIRelationshipOWNER,
		},
		{Key: &run_model.APIResourceKey{Type: run_model.APIResourceTypeJOB, ID: jobID},
			Name: jobName, Relationship: run_model.APIRelationshipCREATOR,
		},
	}
	assert.Equal(t, resourceReferences, run.ResourceReferences)
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
	test.DeleteAllExperiments(s.experimentClient, s.T())
	test.DeleteAllPipelines(s.pipelineClient, s.T())
	test.DeleteAllJobs(s.jobClient, s.T())
	test.DeleteAllRuns(s.runClient, s.T())
}

func defaultApiJob(pipelineVersionId, experimentId string) *job_model.APIJob {
	return &job_model.APIJob{
		Name:        "default-pipeline-name",
		Description: "This is a default pipeline",
		ResourceReferences: []*job_model.APIResourceReference{
			{Key: &job_model.APIResourceKey{Type: job_model.APIResourceTypeEXPERIMENT, ID: experimentId},
				Relationship: job_model.APIRelationshipOWNER},
			{Key: &job_model.APIResourceKey{Type: job_model.APIResourceTypePIPELINEVERSION, ID: pipelineVersionId},
				Relationship: job_model.APIRelationshipCREATOR},
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
